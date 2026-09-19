# jevium

Jev chooses an observed action. Code owns execution. Chrome (desktop CDP) and Appium (iPhone XCUITest) are the two surfaces.

Give it one goal. TypeSafe Jev picks an operation and an element from a structured action table. A small LLM writes text only when the operation is `TYPE_TEXT`. Model output never becomes selectors, coordinates, or executable code.

```text
observe → indexed actions → one TypeSafe request → execute
```

`DONE` is not success. Verify the resulting page or accessibility tree independently.

## Setup

```bash
cd ~/dev/jevium
go test ./...
```

`TYPESAFE_API_KEY` is read from the process environment. A local `.env` only fills keys that are still empty. Do not pass `--env-file` tools that overwrite an exported key with a blank value. `TEXT_MODEL_API_KEY` is required only when a field must be typed.

## Appium (iPhone)

Appium must already be running on `http://127.0.0.1:4723`. Set `APPIUM_UDID` (or pass `--udid`). There is no default device.

```bash
go run ./cmd/jevium --mode appium \
  --udid "$APPIUM_UDID" \
  --bundle-id com.apple.springboard \
  --goal 'Open Safari and load https://example.com. Stop when Example Domain is visible.'
```

`--bundle-id com.apple.mobilesafari` starts in Safari. `--tui` opens a Bubble Tea inspector.

## Chrome

Start Chrome with remote debugging, then:

```bash
# example
# /Applications/Google\ Chrome.app/Contents/MacOS/Google\ Chrome --remote-debugging-port=9222

go run ./cmd/jevium --mode chrome \
  --url 'https://example.com' \
  --goal 'Stop when Example Domain is visible.'
```

Safari on a phone is Appium (`com.apple.mobilesafari`). Desktop Chrome is CDP.

## Library

```go
device, err := appium.New(appium.Config{UDID: udid, BundleID: "com.apple.mobilesafari"})
agent, err := agent.New(device, policy.Client{}, goal, false, "")
err = agent.Run()
```
