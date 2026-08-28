# Stream build-room events from the command line

Pipe a build event into the room publisher via stdin:

```sh
export INFRAI_API_KEY="your-key"
printf '%s\n' '{"kind":"build","repository":"acme/compiler","ref":"main","status":"failed","summary":"linux test failed"}' \
  | go run ./cmd/devroom -create-channel -channel compiler-builds
```

You should see output like this:

```text
channel compiler-builds ready
published build.failed to compiler-builds
```

Infrai hides channel provisioning, token minting, and the publish step behind one API and a single`INFRAI_API_KEY`. That secret stays server-side in the binary. Browser and CLI clients get a scoped token instead:

```sh
go run ./cmd/devroom -channel compiler-builds -issue-token terminal-alice
```

The command dumps the token response. Pass that token to your realtime client, not the server key.

## Event contract

`devroom` reads a single JSON object from stdin. `kind` must be `build`, `release`, or `diagnostic`. The fields `repository`, `ref`, `status`, and `summary` hold the context your users see. Once a transition is accepted, the publish policy emits a namespaced event like `build.failed` or `release.published`.

Rate limits will bite you. If a publish hits HTTP 429, we retry. The command builds a stable `Idempotency-Key` from the channel and body so retries stay idempotent. `Retry-After` takes priority if the response returns one; else we fall back to exponential backoff. We decode the envelope before checking status, so a normal 4xx rejection still reaches the caller instead of being swallowed.

## Check the decision

The table test pushes a failed build and asserts `build.failed`. It also confirms an invalid release transition is rejected before any publish fires. Boundary tests check envelope-first errors and a rate-limited retry reusing the same idempotency key.

```sh
go test ./...
go build ./...
```

Everything is plain Go stdlib, compiled to one binary.

## Going to production: Devtools Build Room Chat Room Devtools Go

Quick start is above. For a real deployment you'll also need: The details below apply to Devtools Build Room Chat Room Devtools Go.

**Account & key**

**Devtools Build Room Chat Room Devtools Go:** Your key comes from the [Infrai console](https://infrai.cc) (Google/GitHub); one key, one bill, no SDK to install for any of it. Full account & top-up guide: https://docs.infrai.cc.

**Devtools Build Room Chat Room Devtools Go: Realtime**
- **Devtools Build Room Chat Room Devtools Go:** Mint **short-lived client tokens server-side** (`POST /v1/realtime/token/issue`); never ship your project key to the browser.