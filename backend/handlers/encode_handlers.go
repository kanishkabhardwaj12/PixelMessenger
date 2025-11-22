package handlers

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"strings"
	"log"
	"time"

	"github.com/google/uuid"
	ws "github.com/kanishkabhardwaj12/PixelMessenger/backend/websocket"
	"github.com/kanishkabhardwaj12/PixelMessenger/backend/models"
	"github.com/kanishkabhardwaj12/PixelMessenger/backend/storage"
)

// EncodeRequest is the JSON body accepted by /encode
type EncodeRequest struct {
	ImageBase64 string `json:"image_base64"`
	Message     string `json:"message"`
	RoomID      string `json:"room_id"`
	Passphrase  string `json:"passphrase,omitempty"`
}

// EncodeImage returns an HTTP handler that accepts a base64 image + message,
// embeds the message into the provided image and broadcasts the resulting PNG
// to the requested room via the Hub. It also returns the encoded image and
// decoded message in the HTTP response.
func EncodeImage(hub *ws.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Read body
		var req EncodeRequest
		contentType := r.Header.Get("Content-Type")
		if strings.HasPrefix(contentType, "multipart/form-data") || strings.Contains(contentType, "multipart/form-data") {
			// Parse multipart form
			if err := r.ParseMultipartForm(32 << 20); err != nil {
				http.Error(w, "Failed to parse multipart form: "+err.Error(), http.StatusBadRequest)
				return
			}
			file, _, err := r.FormFile("image")
			if err != nil {
				http.Error(w, "Missing 'image' file in form: "+err.Error(), http.StatusBadRequest)
				return
			}
			defer file.Close()
			imgBytes, err := io.ReadAll(file)
			if err != nil {
				http.Error(w, "Failed to read uploaded image: "+err.Error(), http.StatusBadRequest)
				return
			}
			req.ImageBase64 = base64.StdEncoding.EncodeToString(imgBytes)
			req.Message = r.FormValue("message")
			req.Passphrase = r.FormValue("passphrase")
			req.RoomID = r.FormValue("room_id")
		} else if contentType == "application/json" || strings.Contains(contentType, "application/json") {
			dec := json.NewDecoder(r.Body)
			if err := dec.Decode(&req); err != nil {
				http.Error(w, "Invalid JSON body: "+err.Error(), http.StatusBadRequest)
				return
			}
		} else {
			// Try to read raw body as JSON anyway
			body, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "Failed to read request body: "+err.Error(), http.StatusBadRequest)
				return
			}
			if err := json.Unmarshal(body, &req); err != nil {
				http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
				return
			}
		}

		if req.ImageBase64 == "" || req.Message == "" || req.RoomID == "" {
			http.Error(w, "image_base64, message and room_id are required", http.StatusBadRequest)
			return
		}

		// Support data URLs like "data:image/png;base64,..."
		imgPart := req.ImageBase64
		if strings.HasPrefix(imgPart, "data:") {
			idx := strings.Index(imgPart, ",")
			if idx >= 0 {
				imgPart = imgPart[idx+1:]
			}
		}

		// Decode the provided base64 image and forward to AI service for encoding
		imgBytes, err := base64.StdEncoding.DecodeString(imgPart)
		if err != nil {
			http.Error(w, "Failed to decode base64 image: "+err.Error(), http.StatusBadRequest)
			return
		}

		// Determine AI service URL
		aiURL := os.Getenv("AI_SERVICE_URL")
		if aiURL == "" {
			aiURL = "http://localhost:5000/encode-image"
		} else {
			// If environment provides only base path, ensure endpoint exists
			parsed, _ := url.Parse(aiURL)
			if parsed.Path == "" || parsed.Path == "/" {
				aiURL = strings.TrimRight(aiURL, "/") + "/encode-image"
			}
		}

		// Build multipart form request to AI service
		var body bytes.Buffer
		mw := multipart.NewWriter(&body)
		// image file
		fw, err := mw.CreateFormFile("image", "upload.png")
		if err != nil {
			http.Error(w, "Failed to create multipart: "+err.Error(), http.StatusInternalServerError)
			return
		}
		if _, err := fw.Write(imgBytes); err != nil {
			http.Error(w, "Failed to write image to multipart: "+err.Error(), http.StatusInternalServerError)
			return
		}
		// message field
		if err := mw.WriteField("message", req.Message); err != nil {
			http.Error(w, "Failed to write message field: "+err.Error(), http.StatusInternalServerError)
			return
		}
		// optional passphrase (forward to AI service if provided)
		if req.Passphrase != "" {
			if err := mw.WriteField("passphrase", req.Passphrase); err != nil {
				http.Error(w, "Failed to write passphrase field: "+err.Error(), http.StatusInternalServerError)
				return
			}
		}
		mw.Close()

		aiReq, err := http.NewRequest("POST", aiURL, &body)
		if err != nil {
			http.Error(w, "Failed to create AI request: "+err.Error(), http.StatusInternalServerError)
			return
		}
		aiReq.Header.Set("Content-Type", mw.FormDataContentType())

		client := &http.Client{}
		respAi, err := client.Do(aiReq)
		if err != nil {
			http.Error(w, "AI service request failed: "+err.Error(), http.StatusInternalServerError)
			return
		}
		defer respAi.Body.Close()

		if respAi.StatusCode != http.StatusOK {
			// forward body as error
			bodyBytes, _ := io.ReadAll(respAi.Body)
			http.Error(w, "AI service error: "+string(bodyBytes), http.StatusInternalServerError)
			return
		}

		var aiResp struct {
			EncodedImageBase64 string `json:"encoded_image_base64"`
			DecodedMessage     string `json:"decoded_message"`
		}
		if err := json.NewDecoder(respAi.Body).Decode(&aiResp); err != nil {
			http.Error(w, "Failed to parse AI response: "+err.Error(), http.StatusInternalServerError)
			return
		}

		encodedBytes, err := base64.StdEncoding.DecodeString(aiResp.EncodedImageBase64)
		if err != nil {
			http.Error(w, "Failed to decode AI image base64: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Persist message to DB (so clients can rehydrate later)
		senderID := ""
		if v := r.Context().Value("userID"); v != nil {
			senderID = v.(string)
		}
		// create message record
		msg := models.Message{
			ID:                  uuid.New().String(),
			RoomID:              req.RoomID,
			SenderID:            senderID,
			DecodedText:         aiResp.DecodedMessage,
			EncodedImageBase64:  aiResp.EncodedImageBase64,
			OriginalImageBase64: req.ImageBase64, // Store original image
			CreatedAt:           time.Now().UTC(),
		}
		if err := storage.SaveMessage(msg); err != nil {
			// log error but continue broadcasting
			// (don't fail the request just because DB save failed)
			// Use the standard log package to avoid importing fmt here
			log.Println("Failed to save message to DB:", err)
		}

		// Broadcast to hub with decoded text from AI and original image, including message ID
		hub.PublishEncodedImage(req.RoomID, msg.ID, encodedBytes, senderID, aiResp.DecodedMessage, imgBytes)

		// Return AI response to caller
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"encoded_image":  aiResp.EncodedImageBase64,
			"decoded_message": aiResp.DecodedMessage,
		})
	}
}
