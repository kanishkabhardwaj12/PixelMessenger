package handlers

import (
	"encoding/json"
	"image"
	"net/http"

	"github.com/kanishkabhardwaj12/PixelMessenger/backend/steganography"
	"encoding/base64"
	"unicode/utf8"
)

func DecodeImage(w http.ResponseWriter, r *http.Request) {
	// image to be sent as the request body
	img, _, err := image.Decode(r.Body)
	if err != nil {
		http.Error(w, "Failed to decode image: "+err.Error(), http.StatusBadRequest)
		return
	}

	message, err := steganography.Decode(img)
	if err != nil {
		http.Error(w, "Failed to decode message from image: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Detect our marker. We expect encoded payloads to be prefixed with "B64:"
	// when the sender base64-encoded the original bytes prior to embedding.
	resp := make(map[string]string)
	const marker = "B64:"
	if len(message) >= len(marker) && string(message[:len(marker)]) == marker {
		// Strip marker and base64-decode
		b64part := message[len(marker):]
		decoded, err := base64.StdEncoding.DecodeString(string(b64part))
		if err != nil {
			// If base64 fails, fall back to returning the raw base64 as a string
			resp["encoding"] = "base64"
			resp["message_base64"] = string(b64part)
			// Do NOT set resp["message"] to raw binary text; frontend should use message_base64
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		}

		// Now 'decoded' is the original message bytes. Try to return as UTF-8 if possible.
		if utf8.Valid(decoded) {
			resp["encoding"] = "utf8"
			resp["message"] = string(decoded)
		} else {
			resp["encoding"] = "base64"
			// Return the original bytes as base64 so the frontend can choose how to present them
			resp["message_base64"] = base64.StdEncoding.EncodeToString(decoded)
			// Do NOT set resp["message"] to the raw binary string to avoid mojibake in the UI
		}
	} else {
		// No marker — treat as before: if valid UTF-8, return directly; otherwise return base64
		if utf8.Valid(message) {
			resp["encoding"] = "utf8"
			resp["message"] = string(message)
		} else {
			resp["encoding"] = "base64"
			resp["message_base64"] = base64.StdEncoding.EncodeToString(message)
			resp["message"] = string(message)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
