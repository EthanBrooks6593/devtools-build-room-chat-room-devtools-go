package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	"example.com/devtools-build-room/internal/buildfeed"
	"example.com/devtools-build-room/internal/realtime"
)

func main() {
	channel := flag.String("channel", "compiler-builds", "realtime channel")
	accountID := flag.String("account", "local-ci", "publishing account")
	bootstrap := flag.Bool("create-channel", false, "create the channel before publishing")
	clientID := flag.String("issue-token", "", "issue a subscriber token for this client ID")
	flag.Parse()

	key := os.Getenv("INFRAI_API_KEY")
	if key == "" {
		exitf("INFRAI_API_KEY is required")
	}
	client := realtime.NewClient(key)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if *bootstrap {
		if err := client.CreateChannel(ctx, *channel, "chat", "", "channel-"+*channel); err != nil {
			exitAPI(err)
		}
		fmt.Printf("channel %s ready\n", *channel)
	}
	if *clientID != "" {
		data, err := client.IssueToken(ctx, *clientID, []string{*channel}, []string{"subscribe"}, 3600)
		if err != nil {
			exitAPI(err)
		}
		fmt.Println(string(data))
		return
	}

	var input buildfeed.Event
	if err := json.NewDecoder(os.Stdin).Decode(&input); err != nil {
		exitf("read event: %v", err)
	}
	publication, err := buildfeed.Prepare(input)
	if err != nil {
		exitf("event rejected: %v", err)
	}
	idempotencyKey := eventKey(*channel, publication.Data)
	if err := client.Publish(ctx, *channel, publication.Event, publication.Data, *accountID, idempotencyKey); err != nil {
		exitAPI(err)
	}
	fmt.Printf("published %s to %s\n", publication.Event, *channel)
}

func eventKey(channel string, data []byte) string {
	sum := sha256.Sum256(append([]byte(channel+"\n"), data...))
	return "event-" + hex.EncodeToString(sum[:16])
}

func exitAPI(err error) {
	var apiErr *realtime.APIError
	if errors.As(err, &apiErr) && apiErr.HTTPStatus >= 400 && apiErr.HTTPStatus < 500 {
		fmt.Fprintf(os.Stderr, "request rejected: %s\n", apiErr)
		os.Exit(2)
	}
	exitf("request failed: %v", err)
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
