# Device controls

Pass `--device-controls` in Appium mode to offer rotation, lock/unlock, and keyboard dismissal. Controls derive from current OS lock state and orientation; keyboard dismissal is offered when the native source contains a keyboard. The executor refreshes that state before every action and never retries a mutation.

Unlock delegates to Appium's OS unlock operation. It cannot bypass a passcode. Apps may refuse a requested orientation; inspect the next observation rather than treating command acceptance as proof. Keyboard dismissal can be unsupported by the current app or keyboard and returns the Appium error without a blind tap fallback.
