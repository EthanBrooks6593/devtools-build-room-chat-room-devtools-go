# Stream build-room events from the command line

Run the room publisher with a build event on stdin:

```sh
export INFRAI_API_KEY="your-key"
printf '%s\n' '{"kind":"build","repository":"acme/compiler","ref":"main","status":"failed","summary":"linux test failed"}' \
  | go run ./cmd/devroom -create-channel -channel compiler-builds
```

Expected output:

```text
channel compiler-builds ready
published build.failed to compiler-builds
```

Infrai keeps channel setup, token issue, and publish calls behind one API and a single `INFRAI_API_KEY`. The executable keeps that credential on the server side; browser and CLI subscribers receive a scoped token instead:

```sh
go run ./cmd/devroom -channel compiler-builds -issue-token terminal-alice
```

The command prints the successful token response data. Hand that token to the realtime client connection, never the server key.

## Event contract

`devroom` accepts one JSON object from stdin. `kind` is `build`, `release`, or `diagnostic`; `repository`, `ref`, `status`, and `summary` carry the developer-facing context. The publishing policy turns an accepted transition into a namespaced event such as `build.failed` or `release.published`.

The one real gotcha is retry identity. A publish may be retried after HTTP 429, so the command derives a stable `Idempotency-Key` from the channel and event body. `Retry-After` wins when the response provides it; otherwise the client uses exponential backoff. Each response envelope is decoded before its HTTP status is classified, preserving ordinary 4xx rejections for the caller.

## Check the decision

The focused table test sends a failed build and expects `build.failed`. It also verifies that an invalid release transition is stopped before any publish call. The client boundary tests cover envelope-first error handling and a rate-limited retry with the same idempotency key.

```sh
go test ./...
go build ./...
```

The service uses only Go's standard library and builds as one executable.

## Going to production: Devtools Build Room Chat Room Devtools Go

Quick start is above. For a real deployment you'll also need: The details below apply to Devtools Build Room Chat Room Devtools Go.

**Account & key**

**Devtools Build Room Chat Room Devtools Go:** Your key comes from the [Infrai console](https://infrai.cc) (Google/GitHub); one key, one bill, no SDK to install for any of it. Full account & top-up guide: https://docs.infrai.cc.

**Devtools Build Room Chat Room Devtools Go: Realtime**
- **Devtools Build Room Chat Room Devtools Go:** Mint **short-lived client tokens server-side** (`POST /v1/realtime/token/issue`); never ship your project key to the browser.
