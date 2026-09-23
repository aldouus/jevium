//go:build darwin

package visual

import (
	"os/exec"
	"testing"
)

func TestMacOCRRecognizesRenderedText(t *testing.T) {
	frame, err := exec.Command("xcrun", "swift", "-e", `
import AppKit
let bitmap = NSBitmapImageRep(bitmapDataPlanes: nil, pixelsWide: 800, pixelsHigh: 200, bitsPerSample: 8, samplesPerPixel: 4, hasAlpha: true, isPlanar: false, colorSpaceName: .deviceRGB, bytesPerRow: 0, bitsPerPixel: 0)!
NSGraphicsContext.saveGraphicsState()
NSGraphicsContext.current = NSGraphicsContext(bitmapImageRep: bitmap)
NSColor.white.setFill()
NSRect(x: 0, y: 0, width: 800, height: 200).fill()
("Continue" as NSString).draw(at: NSPoint(x: 50, y: 75), withAttributes: [.font: NSFont.systemFont(ofSize: 64), .foregroundColor: NSColor.black])
NSGraphicsContext.restoreGraphicsState()
FileHandle.standardOutput.write(bitmap.representation(using: .png, properties: [:])!)
`).Output()
	if err != nil {
		t.Fatalf("render fixture: %v", err)
	}
	regions, err := (MacOCR{}).Recognize(frame)
	if err != nil {
		t.Fatal(err)
	}
	if err := Validate(frame, regions); err != nil {
		t.Fatal(err)
	}
	if len(regions) != 1 {
		t.Fatalf("regions=%+v", regions)
	}
	if regions[0].Label != "Continue" {
		t.Fatalf("recognized %q", regions[0].Label)
	}
	if regions[0].X < 0.04 || regions[0].X > 0.09 || regions[0].W < 0.2 {
		t.Fatalf("text bounds=%+v", regions[0])
	}
}
