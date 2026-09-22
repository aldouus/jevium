import Foundation
import Vision
import ImageIO

struct Region: Codable {
    let label: String
    let x: Double
    let y: Double
    let w: Double
    let h: Double
}

do {
    let data = FileHandle.standardInput.readDataToEndOfFile()
    guard let source = CGImageSourceCreateWithData(data as CFData, nil),
          let image = CGImageSourceCreateImageAtIndex(source, 0, nil) else {
        throw NSError(domain: "jevium.ocr", code: 1, userInfo: [NSLocalizedDescriptionKey: "Invalid screenshot"])
    }
    let request = VNRecognizeTextRequest()
    request.recognitionLevel = .accurate
    request.usesLanguageCorrection = false
    try VNImageRequestHandler(cgImage: image).perform([request])
    let regions = (request.results ?? []).compactMap { observation -> Region? in
        guard let text = observation.topCandidates(1).first, text.confidence >= 0.8 else { return nil }
        let rect = observation.boundingBox
        return Region(label: text.string, x: rect.minX, y: 1 - rect.maxY, w: rect.width, h: rect.height)
    }
    FileHandle.standardOutput.write(try JSONEncoder().encode(regions))
} catch {
    FileHandle.standardError.write(Data("OCR failed: \(error.localizedDescription)\n".utf8))
    exit(1)
}
