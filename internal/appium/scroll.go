package appium

import "github.com/aldous/jevium/internal/page"

func containsRect(parent, child page.Rect) bool {
	return child.X >= parent.X && child.Y >= parent.Y && child.X+child.W <= parent.X+parent.W && child.Y+child.H <= parent.Y+parent.H
}

// Choose an exposed strip so scrolling a parent does not start in its child.
func outsideRect(parent, child page.Rect) []page.Rect {
	overlap := intersect(child, &parent)
	if overlap.W <= 0 || overlap.H <= 0 {
		return []page.Rect{parent}
	}
	candidates := []page.Rect{
		{X: parent.X, Y: parent.Y, W: overlap.X - parent.X, H: parent.H},
		{X: overlap.X + overlap.W, Y: parent.Y, W: parent.X + parent.W - overlap.X - overlap.W, H: parent.H},
		{X: overlap.X, Y: parent.Y, W: overlap.W, H: overlap.Y - parent.Y},
		{X: overlap.X, Y: overlap.Y + overlap.H, W: overlap.W, H: parent.Y + parent.H - overlap.Y - overlap.H},
	}
	var regions []page.Rect
	for _, r := range candidates {
		if r.W >= 12 && r.H >= 24 {
			regions = append(regions, r)
		}
	}
	return regions
}
