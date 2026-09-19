# jevium

Keep the loop small: observe → indexed elements → operation + target → execute.

- One natural-language goal. Do not add site- or app-specific plans or hardcoded field values.
- TypeSafe chooses an operation and operation-specific target heads in one request. Consume only the selected operation's target.
- Targets must map to observed elements. Never let the model emit selectors, coordinates, or executable code.
- TYPE_TEXT invokes the text LLM. Cache a stale retry's value only while its entire helper input is identical.
- Never retry a mutation. Log execution before observing its result.
- Screenshots are optional; Jev does not consume them.
- Keep credentials in the process environment or `.env`. Tests must not call paid APIs or a live phone.
- Verify actual final outcomes independently. A DONE choice is not proof of success.

Checks: `go test ./...`
