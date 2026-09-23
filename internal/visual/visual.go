package visual

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

type Region struct {
	Label string  `json:"label"`
	X     float64 `json:"x"`
	Y     float64 `json:"y"`
	W     float64 `json:"w"`
	H     float64 `json:"h"`
}

type Recognizer interface {
	Recognize([]byte) ([]Region, error)
}

//go:embed ocr.swift
var swiftSource string

type MacOCR struct{}

func (MacOCR) Recognize(frame []byte) ([]Region, error) {
	if runtime.GOOS != "darwin" {
		return nil, fmt.Errorf("visual OCR requires macOS Vision and Xcode command line tools")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "xcrun", "swift", "-e", swiftSource)
	cmd.Stdin = bytes.NewReader(frame)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	data, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("macOS Vision OCR failed: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	var regions []Region
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&regions); err != nil {
		return nil, fmt.Errorf("invalid OCR result: %w", err)
	}
	return regions, nil
}

func Validate(frame []byte, regions []Region) error {
	cfg, _, err := image.DecodeConfig(bytes.NewReader(frame))
	if err != nil {
		return fmt.Errorf("invalid visual screenshot: %w", err)
	}
	if cfg.Width <= 0 || cfg.Height <= 0 || cfg.Width > 20000 || cfg.Height > 20000 {
		return fmt.Errorf("invalid visual screenshot dimensions")
	}
	if len(regions) > 250 {
		return fmt.Errorf("too many OCR regions")
	}
	for _, r := range regions {
		if strings.TrimSpace(r.Label) == "" || len(r.Label) > 1000 {
			return fmt.Errorf("invalid OCR label")
		}
		for _, n := range []float64{r.X, r.Y, r.W, r.H} {
			if math.IsNaN(n) || math.IsInf(n, 0) {
				return fmt.Errorf("invalid OCR geometry")
			}
		}
		if r.X < 0 || r.Y < 0 || r.W <= 0 || r.H <= 0 || r.X+r.W > 1 || r.Y+r.H > 1 {
			return fmt.Errorf("OCR region lies outside screenshot")
		}
	}
	return nil
}
