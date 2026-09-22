package appium

import "github.com/aldous/jevium/internal/page"

func containsRect(parent, child page.Rect) bool {
	return child.X >= parent.X && child.Y >= parent.Y && child.X+child.W <= parent.X+parent.W && child.Y+child.H <= parent.Y+parent.H
}

// Choose an exposed strip so scrolling a parent does not start in its child.
func outsideRect(parent, child page.Rect) page.Rect {
	overlap := intersect(child, &parent)
	if overlap.W <= 0 || overlap.H <= 0 {
		return parent
	}
	candidates := []page.Rect{
		{X: parent.X, Y: parent.Y, W: overlap.X - parent.X, H: parent.H},
		{X: overlap.X + overlap.W, Y: parent.Y, W: parent.X + parent.W - overlap.X - overlap.W, H: parent.H},
		{X: parent.X, Y: parent.Y, W: parent.W, H: overlap.Y - parent.Y},
		{X: parent.X, Y: overlap.Y + overlap.H, W: parent.W, H: parent.Y + parent.H - overlap.Y - overlap.H},
	}
	var best page.Rect
	for _, r := range candidates {
		if r.W*r.H > best.W*best.H {
			best = r
		}
	}
	return best
}
