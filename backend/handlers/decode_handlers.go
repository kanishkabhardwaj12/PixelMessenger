package handlers

import (
	"encoding/json"
	"image"
	"net/http"

	"github.com/kanishkabhardwaj12/PixelMessenger/backend/steganography"
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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": string(message),
	})
}
