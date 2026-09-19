# jevium

Jevium is a harness that runs a natural language UI goal by letting Jev pick the next tap or keystroke from what is on screen, then executing it with Appium or Chrome.

```bash
export APPIUM_UDID=

./jevium --mode appium --bundle-id com.apple.mobilesafari --goal 'Open https://www.google.com and type "hello world". Stop when search results are visible.'

./jevium --mode chrome --url 'https://www.google.com' --goal 'Type "hello world" in the search box. Stop when results are visible.'
```
