package visual

import (
	"bytes"
	"image"
	"image/png"
	"math"
	"strings"
	"testing"
)

func TestValidateObservation(t *testing.T) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 10, 10))); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		region Region
		want   string
	}{
		{Region{Label: "", W: 0.1, H: 0.1}, "invalid OCR label"},
		{Region{Label: "button", X: math.NaN(), W: 0.1, H: 0.1}, "invalid OCR geometry"},
		{Region{Label: "button", X: 0.9, W: 0.2, H: 0.1}, "OCR region lies outside screenshot"},
	} {
		if err := Validate(buf.Bytes(), []Region{tc.region}); err == nil || err.Error() != tc.want {
			t.Fatalf("error=%v want=%q", err, tc.want)
		}
	}
	for _, frame := range [][]byte{nil, []byte("not image")} {
		if err := Validate(frame, nil); err == nil || !strings.HasPrefix(err.Error(), "invalid visual screenshot:") {
			t.Fatalf("error=%v", err)
		}
	}
}
