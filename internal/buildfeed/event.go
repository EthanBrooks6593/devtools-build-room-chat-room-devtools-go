package buildfeed

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Event is the input accepted by the devroom command.
type Event struct {
	Kind       string `json:"kind"`
	Repository string `json:"repository"`
	Ref        string `json:"ref"`
	Status     string `json:"status"`
	Summary    string `json:"summary"`
}

// Publication is the concrete realtime message produced by an accepted event.
type Publication struct {
	Event string
	Data  json.RawMessage
}

// Prepare applies the room's publishing policy and encodes the resulting message.
func Prepare(input Event) (Publication, error) {
	if strings.TrimSpace(input.Repository) == "" {
		return Publication{}, fmt.Errorf("repository is required")
	}

	allowed := map[string]map[string]bool{
		"build":      {"started": true, "passed": true, "failed": true},
		"release":    {"started": true, "published": true},
		"diagnostic": {"info": true, "warning": true, "error": true},
	}
	statuses, knownKind := allowed[input.Kind]
	if !knownKind || !statuses[input.Status] {
		return Publication{}, fmt.Errorf("unsupported %s status %q", input.Kind, input.Status)
	}

	payload, err := json.Marshal(input)
	if err != nil {
		return Publication{}, fmt.Errorf("encode event: %w", err)
	}
	return Publication{Event: input.Kind + "." + input.Status, Data: payload}, nil
}
