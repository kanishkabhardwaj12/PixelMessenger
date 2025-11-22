package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
)

// AnalysisSteganography compares a cover image with a stego image and returns
// detailed statistics about the pixel differences, specifically focusing on
// LSB (Least Significant Bit) changes used in steganography.
func AnalyzeSteganography(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse multipart form (expecting cover_image and stego_image)
	if err := r.ParseMultipartForm(32 << 20); err != nil { // 32MB max
		http.Error(w, "Failed to parse multipart form: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Get cover image
	coverFile, _, err := r.FormFile("cover_image")
	if err != nil {
		http.Error(w, "Missing 'cover_image' file: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer coverFile.Close()

	// Get stego image
	stegoFile, _, err := r.FormFile("stego_image")
	if err != nil {
		http.Error(w, "Missing 'stego_image' file: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer stegoFile.Close()

	// Read cover image
	coverBytes, err := io.ReadAll(coverFile)
	if err != nil {
		http.Error(w, "Failed to read cover image: "+err.Error(), http.StatusBadRequest)
		return
	}
	coverImg, _, err := image.Decode(bytes.NewReader(coverBytes))
	if err != nil {
		http.Error(w, "Failed to decode cover image: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Read stego image
	stegoBytes, err := io.ReadAll(stegoFile)
	if err != nil {
		http.Error(w, "Failed to read stego image: "+err.Error(), http.StatusBadRequest)
		return
	}
	stegoImg, _, err := image.Decode(bytes.NewReader(stegoBytes))
	if err != nil {
		http.Error(w, "Failed to decode stego image: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Perform analysis
	analysis := analyzeImages(coverImg, stegoImg)

	// Return JSON response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(analysis)
}

// analyzeImages compares two images and returns detailed statistics
func analyzeImages(cover, stego image.Image) map[string]interface{} {
	coverBounds := cover.Bounds()
	stegoBounds := stego.Bounds()

	// Check if dimensions match
	if coverBounds != stegoBounds {
		return map[string]interface{}{
			"error": "Image dimensions do not match",
			"cover_dimensions": fmt.Sprintf("%dx%d", coverBounds.Dx(), coverBounds.Dy()),
			"stego_dimensions": fmt.Sprintf("%dx%d", stegoBounds.Dx(), stegoBounds.Dy()),
		}
	}

	width := coverBounds.Dx()
	height := coverBounds.Dy()
	totalPixels := width * height
	totalChannels := totalPixels * 3 // R, G, B

	// Statistics
	var totalChanges int
	var lsbChanges int
	var maxDifference uint32
	samplePixels := make([]map[string]interface{}, 0)
	binaryComparison := make([]string, 0)

	// Sample first 10 pixels for detailed comparison
	sampleCount := 0
	maxSamples := 10

	// Iterate through all pixels
	for y := coverBounds.Min.Y; y < coverBounds.Max.Y; y++ {
		for x := coverBounds.Min.X; x < coverBounds.Max.X; x++ {
			coverColor := cover.At(x, y)
			stegoColor := stego.At(x, y)

			coverR, coverG, coverB, _ := coverColor.RGBA()
			stegoR, stegoG, stegoB, _ := stegoColor.RGBA()

			// Convert from 16-bit to 8-bit
			cr := uint8(coverR >> 8)
			cg := uint8(coverG >> 8)
			cb := uint8(coverB >> 8)
			sr := uint8(stegoR >> 8)
			sg := uint8(stegoG >> 8)
			sb := uint8(stegoB >> 8)

			// Check for differences
			rDiff := absDiff(cr, sr)
			gDiff := absDiff(cg, sg)
			bDiff := absDiff(cb, sb)

			// Count any changes
			if rDiff > 0 {
				totalChanges++
				if rDiff == 1 {
					lsbChanges++
				}
			}
			if gDiff > 0 {
				totalChanges++
				if gDiff == 1 {
					lsbChanges++
				}
			}
			if bDiff > 0 {
				totalChanges++
				if bDiff == 1 {
					lsbChanges++
				}
			}

			// Track max difference
			if uint32(rDiff) > maxDifference {
				maxDifference = uint32(rDiff)
			}
			if uint32(gDiff) > maxDifference {
				maxDifference = uint32(gDiff)
			}
			if uint32(bDiff) > maxDifference {
				maxDifference = uint32(bDiff)
			}

			// Sample first few changed pixels for detailed view
			if sampleCount < maxSamples && (rDiff > 0 || gDiff > 0 || bDiff > 0) {
				samplePixels = append(samplePixels, map[string]interface{}{
					"position": fmt.Sprintf("(%d, %d)", x, y),
					"cover":    fmt.Sprintf("RGB(%d, %d, %d)", cr, cg, cb),
					"stego":    fmt.Sprintf("RGB(%d, %d, %d)", sr, sg, sb),
					"diff":     fmt.Sprintf("R:%d G:%d B:%d", rDiff, gDiff, bDiff),
				})

				// Binary representation for first 5 changed pixels
				if sampleCount < 5 {
					binaryComparison = append(binaryComparison,
						fmt.Sprintf("Pixel (%d,%d):", x, y),
						fmt.Sprintf("  Cover R: %08b (%d) → Stego R: %08b (%d) [LSB: %d→%d]",
							cr, cr, sr, sr, cr&1, sr&1),
						fmt.Sprintf("  Cover G: %08b (%d) → Stego G: %08b (%d) [LSB: %d→%d]",
							cg, cg, sg, sg, cg&1, sg&1),
						fmt.Sprintf("  Cover B: %08b (%d) → Stego B: %08b (%d) [LSB: %d→%d]",
							cb, cb, sb, sb, cb&1, sb&1),
						"",
					)
				}

				sampleCount++
			}
		}
	}

	// Calculate capacity and usage
	capacityBytes := totalChannels / 8
	lsbChangedBytes := lsbChanges / 8
	capacityUsedPercent := 0.0
	if capacityBytes > 0 {
		capacityUsedPercent = float64(lsbChangedBytes) / float64(capacityBytes) * 100.0
	}

	// Build sample pixels string
	samplePixelsStr := ""
	for _, sample := range samplePixels {
		samplePixelsStr += fmt.Sprintf("Position: %s\n", sample["position"])
		samplePixelsStr += fmt.Sprintf("  Cover: %s\n", sample["cover"])
		samplePixelsStr += fmt.Sprintf("  Stego: %s\n", sample["stego"])
		samplePixelsStr += fmt.Sprintf("  Diff:  %s\n\n", sample["diff"])
	}

	// Build binary comparison string
	binaryComparisonStr := ""
	for _, line := range binaryComparison {
		binaryComparisonStr += line + "\n"
	}

	return map[string]interface{}{
		"total_changes":      totalChanges,
		"lsb_changes":        lsbChanges,
		"max_difference":     maxDifference,
		"dimensions":         fmt.Sprintf("%dx%d", width, height),
		"total_pixels":       totalPixels,
		"total_channels":     totalChannels,
		"capacity_bytes":     capacityBytes,
		"capacity_used":      fmt.Sprintf("%d bytes (%.2f%%)", lsbChangedBytes, capacityUsedPercent),
		"sample_pixels":      samplePixelsStr,
		"binary_comparison":  binaryComparisonStr,
		"steganography_detected": lsbChanges > 0,
	}
}

// absDiff returns the absolute difference between two uint8 values
func absDiff(a, b uint8) uint8 {
	if a > b {
		return a - b
	}
	return b - a
}
