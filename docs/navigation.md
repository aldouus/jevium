# iPhone navigation

`--url https://example.com --bundle-id com.apple.mobilesafari` opens the configured URL once before the goal, including when attaching to an existing session. Deep links are not retried on failure.

Repeat `--allow-app com.example.app` to expose explicit launch/switch and terminate controls. A restart uses two decisions: terminate, observe, then launch. This keeps each mutation independently logged; termination is never replayed automatically.

Safari Back, Forward, tab switching, and closing tabs use the existing observed, hittable accessibility targets. Disabled or inaccessible browser controls are not synthesized. App launch does not install missing apps, and native custom-scheme deep links are not supported by this HTTP(S)-only option.
