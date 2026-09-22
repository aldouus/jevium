package appium

import (
	"fmt"
	"github.com/aldous/jevium/internal/page"
)

func gestureActions(actions []page.Action) []page.Action {
	if len(actions) >= page.MaxElementActions {
		return actions
	}
	var gestures []page.Action
	var dragSources []page.Action
	for _, a := range actions {
		if a.Kind != "click" {
			continue
		}
		for _, kind := range []string{"long_press", "double_tap", "swipe_left", "swipe_right", "pinch_in", "pinch_out"} {
			g := a
			g.Kind = kind
			gestures = append(gestures, g)
		}
		if len(dragSources) < 12 && (a.Role == "button" || a.Role == "element") {
			dragSources = append(dragSources, a)
		}
	}
	for i, from := range dragSources {
		for j, to := range dragSources {
			if i == j {
				continue
			}
			g := from
			g.Kind = "drag"
			g.Label = from.Label + " → " + to.Label
			g.Node = fmt.Sprint(from.Node) + "->" + fmt.Sprint(to.Node)
			g.Destination = to.Rect
			gestures = append(gestures, g)
		}
	}
	remaining := page.MaxElementActions - len(actions)
	if len(gestures) > remaining {
		gestures = gestures[:remaining]
	}
	return append(actions, gestures...)
}

func (d *Device) gesture(a page.Action) error {
	x, y, err := tapPoint(a)
	if err != nil {
		return err
	}
	r := a.Rect
	switch a.Kind {
	case "long_press":
		return d.execute("mobile: touchAndHold", []map[string]any{{"x": x, "y": y, "duration": 1}})
	case "double_tap":
		return d.execute("mobile: doubleTap", []point{{X: x, Y: y}})
	case "swipe_left", "swipe_right", "slider":
		from, to := r.X+r.W*.8, r.X+r.W*.2
		if a.Kind == "swipe_right" {
			from, to = to, from
		}
		if a.Kind == "slider" {
			from = x
			to = r.X + r.W*(.1+.8*a.Delta)
		}
		return d.execute("mobile: dragFromToForDuration", []dragArg{{FromX: from, FromY: y, ToX: to, ToY: y, Duration: .1}})
	case "drag":
		if a.Destination == nil || a.Destination.W <= 0 || a.Destination.H <= 0 {
			return StalePageError{Msg: "Drag destination is absent"}
		}
		return d.execute("mobile: dragFromToForDuration", []dragArg{{FromX: x, FromY: y, ToX: a.Destination.X + a.Destination.W/2, ToY: a.Destination.Y + a.Destination.H/2, Duration: 1}})
	case "pinch_in", "pinch_out":
		// Two bounded touch paths avoid pinching the application instead of the target.
		start, end := .15, .35
		if a.Kind == "pinch_in" {
			start, end = end, start
		}
		sources := []map[string]any{}
		for i, sign := range []float64{-1, 1} {
			sources = append(sources, map[string]any{"type": "pointer", "id": fmt.Sprintf("finger%d", i), "parameters": map[string]string{"pointerType": "touch"}, "actions": []map[string]any{
				{"type": "pointerMove", "duration": 0, "origin": "viewport", "x": x + sign*r.W*start, "y": y},
				{"type": "pointerDown", "button": 0},
				{"type": "pointerMove", "duration": 500, "origin": "viewport", "x": x + sign*r.W*end, "y": y},
				{"type": "pointerUp", "button": 0},
			}})
		}
		return d.call("POST", d.path("/actions"), map[string]any{"actions": sources}, nil)
	}
	return UnsupportedError{Msg: "Unsupported gesture"}
}
