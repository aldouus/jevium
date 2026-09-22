# Local visual text targets

`--visual-ocr` enables macOS Vision OCR for Appium observations. It requires macOS and Xcode command line tools (`xcrun swift`). Images stay on this Mac; TypeSafe still receives only indexed text labels. The default observation pipeline is unchanged.

The deterministic OCR adapter supplies normalized text rectangles. Jevium validates image dimensions, labels, finite coordinates, and bounds before projecting them into observed logical screen dimensions. Native click targets take precedence over overlapping OCR text. OCR text regions are labeled as having unknown clickability; they can expose text drawn in a canvas but do not prove that text is a button.

Before execution, both the native snapshot and exact screenshot bytes must still match. A changed frame refuses the action and triggers observation again. Animation, blinking cursors, or changing status-bar pixels can prevent progress; there is no coordinate fallback or mutation retry. This conservative initial implementation does not discover unlabeled icons, arbitrary shapes, or non-text canvas targets.
