package handlers

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"image"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/kanishkabhardwaj12/PixelMessenger/backend/steganography"
)

func DecodeImage(w http.ResponseWriter, r *http.Request) {
	imgBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read image: "+err.Error(), http.StatusBadRequest)
		return
	}

	var goMessage []byte
	img, _, err := image.Decode(bytes.NewReader(imgBytes))
	if err == nil {
		goMessage, _ = steganography.Decode(img)
	}

	const marker = "B64:"
	if len(goMessage) >= len(marker) && string(goMessage[:len(marker)]) == marker {
		writeDecodedMessage(w, goMessage)
		return
	}

	if aiResp, err := decodeWithAIService(imgBytes); err == nil && aiResp != "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"encoding": "utf8",
			"message":  aiResp,
		})
		return
	}

	if goMessage != nil {
		writeDecodedMessage(w, goMessage)
		return
	}

	http.Error(w, "Failed to decode message from image", http.StatusInternalServerError)
}

func writeDecodedMessage(w http.ResponseWriter, message []byte) {
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

func decodeWithAIService(imgBytes []byte) (string, error) {
	aiURL := os.Getenv("AI_SERVICE_URL")
	if aiURL == "" {
		aiURL = "http://localhost:5000/decode-image"
	} else {
		parsed, _ := url.Parse(aiURL)
		if strings.HasSuffix(parsed.Path, "/encode-image") {
			aiURL = strings.TrimSuffix(aiURL, "/encode-image") + "/decode-image"
		} else if parsed.Path == "" || parsed.Path == "/" {
			aiURL = strings.TrimRight(aiURL, "/") + "/decode-image"
		}
	}

	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	fw, err := mw.CreateFormFile("image", "stego.png")
	if err != nil {
		return "", err
	}
	if _, err := fw.Write(imgBytes); err != nil {
		return "", err
	}
	if err := mw.Close(); err != nil {
		return "", err
	}

	req, err := http.NewRequest(http.MethodPost, aiURL, &body)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", io.ErrUnexpectedEOF
	}

	var payload struct {
		DecodedMessage string `json:"decoded_message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", err
	}
	return payload.DecodedMessage, nil
}
