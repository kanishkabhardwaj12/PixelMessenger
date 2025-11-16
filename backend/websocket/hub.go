package ws

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"image"
	"log"
	"os"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/kanishkabhardwaj12/PixelMessenger/backend/steganography"
	"github.com/kanishkabhardwaj12/PixelMessenger/backend/storage"
)

// BroadcastMessage is a TEXT message from a client to the Hub.
type BroadcastMessage struct {
	RoomID  string
	UserID  string
	Message []byte // This is the raw text message
}

// roomBroadcast is an IMAGE message from a worker to the Hub.
// This is a new internal message type.
type roomBroadcast struct {
	RoomID      string
	EncodedData []byte // This is the final encoded PNG
	SenderID    string // user who triggered this broadcast (do not resend to them)
	DecodedText string // The decoded message extracted from the payload
	Timestamp   string // RFC3339 timestamp
}

type Hub struct {
	Rooms      map[string]map[*Client]bool
	Broadcast  chan *BroadcastMessage // Receives TEXT from clients
	Register   chan *Client
	Unregister chan *Client

	// This is the new channel. Workers send the FINAL image here.
	roomBroadcast chan *roomBroadcast
}

func NewHub() *Hub {
	return &Hub{
		Broadcast:     make(chan *BroadcastMessage),
		Register:      make(chan *Client),
		Unregister:    make(chan *Client),
		Rooms:         make(map[string]map[*Client]bool),
		roomBroadcast: make(chan *roomBroadcast), // Initialize the new channel
	}
}

// PublishEncodedImage allows external callers to submit a ready-encoded PNG
// which the hub will broadcast into the given room. This is used by handlers
// that accept user-uploaded images and want the hub to deliver them.
func (h *Hub) PublishEncodedImage(roomID string, data []byte, senderID string, decodedText string) {
	h.roomBroadcast <- &roomBroadcast{
		RoomID:      roomID,
		EncodedData: data,
		SenderID:    senderID,
		DecodedText: decodedText,
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
	}
}

// --- Helper functions (getBestImageURL, fetchImage) are unchanged ---

var sampleImageURLs = []string{
	"https://images.unsplash.com/photo-1500964757637-c85e8a162699?ixlib=rb-4.0.3&q=80&fm=jpg&w=1080",
	"https://images.unsplash.com/photo-1472214103451-9374bd1c798e?ixlib=rb-4.0.3&q=80&fm=jpg&w=1080",
	"https://images.unsplash.com/photo-1470071459604-3b5ec3a7fe05?ixlib=rb-4.0.3&q=80&fm=jpg&w=1080",
	"https://images.unsplash.com/photo-1447752875215-b2761acb3c5d?ixlib=rb-4.0.3&q=80&fm=jpg&w=1080",
	"https://images.unsplash.com/photo-1469474968028-56623f02e42e?ixlib=rb-4.0.3&q=80&fm=jpg&w=1080",
}

func getBestImageURL(aiServiceURL string) string {
	payload, _ := json.Marshal(map[string][]string{
		"image_urls": sampleImageURLs,
	})
	reqBody := bytes.NewBuffer(payload)

	resp, err := http.Post(aiServiceURL, "application/json", reqBody)
	if err != nil {
		log.Printf("AI service call failed: %v", err)
		return sampleImageURLs[0]
	}
	defer resp.Body.Close()

	var result map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Printf("Failed to decode AI response: %v", err)
		return sampleImageURLs[0]
	}

	log.Printf("AI Service selected: %s", result["best_image_url"])
	return result["best_image_url"]
}

func fetchImage(url string) (image.Image, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	img, _, err := image.Decode(resp.Body)
	return img, err
}

// processSteganography is our new "worker" function.
// It runs in its own goroutine and does all the slow work.
func (h *Hub) processSteganography(message *BroadcastMessage) {
	// Allow AI service URL to be configured via environment variable.
	// Fallback to the local development URL if not set.
	aiServiceURL := os.Getenv("AI_SERVICE_URL")
	if aiServiceURL == "" {
		aiServiceURL = "http://localhost:5000/select-image"
	}

	// 1. Get the best image URL from the AI service
	bestImageURL := getBestImageURL(aiServiceURL)

	// 2. Fetch the image from the URL
	img, err := fetchImage(bestImageURL)
	if err != nil {
		log.Printf("Failed to fetch image: %v", err)
		return // Don't send anything if this fails
	}

	// 3. Encode the message into the image
	// To ensure the hidden payload is always valid UTF-8 and safe across languages,
	// encode the message bytes as base64 ASCII before embedding and prefix with a
	// small marker so the decoder can detect and reverse it.
	b64msg := base64.StdEncoding.EncodeToString(message.Message)
	payload := "B64:" + b64msg
	encodedImageBuf, err := steganography.Encode(img, []byte(payload))
	if err != nil {
		log.Printf("Failed to encode steganography: %v", err)
		return // Don't send anything if this fails
	}

	// 4. Send the FINAL encoded image back to the Hub's new channel
	// We also include the decoded text and a timestamp so the hub can broadcast
	// a JSON message containing both the encoded image (base64) and the decoded text.
	decodedText := string(message.Message)
	log.Printf("Worker: encoded image for room=%s sender=%s size=%d bytes", message.RoomID, message.UserID, encodedImageBuf.Len())
	h.roomBroadcast <- &roomBroadcast{
		RoomID:      message.RoomID,
		EncodedData: encodedImageBuf.Bytes(),
		SenderID:    message.UserID,
		DecodedText: decodedText,
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			if h.Rooms[client.RoomID] == nil {
				h.Rooms[client.RoomID] = make(map[*Client]bool)
			}
			h.Rooms[client.RoomID][client] = true
			log.Printf("Client %s registered to room %s", client.UserID, client.RoomID)

		case client := <-h.Unregister:
			if _, ok := h.Rooms[client.RoomID]; ok {
				delete(h.Rooms[client.RoomID], client)
				if len(h.Rooms[client.RoomID]) == 0 {
					delete(h.Rooms, client.RoomID)
				}
				close(client.Send)
				log.Printf("Client %s unregistered from room %s", client.UserID, client.RoomID)
			}

		case message := <-h.Broadcast:
			// A client sent a text message.
			// DO NOT do the work here.
			// Spin off a new goroutine to handle it.
			log.Printf("Broadcast received for room %s, delegating to worker", message.RoomID)
			go h.processSteganography(message)

		case imageMsg := <-h.roomBroadcast:
			// A worker has finished encoding an image.
			// THIS is now the new, fast broadcast logic.
			if clients, ok := h.Rooms[imageMsg.RoomID]; ok {
				log.Printf("Hub: broadcasting image to room=%s sender=%s connections=%d", imageMsg.RoomID, imageMsg.SenderID, len(clients))

				// Try to look up a human-friendly sender name. Fall back to the
				// raw sender id if lookup fails.
				senderName := imageMsg.SenderID
				if u, err := storage.GetUserByID(imageMsg.SenderID); err == nil && u != nil {
					senderName = u.Username
				}

				// Prepare a JSON payload containing base64 image + decoded text + metadata
				payload := map[string]interface{}{
					"type":         "image",
					"image_base64": base64.StdEncoding.EncodeToString(imageMsg.EncodedData),
					"decoded_text": imageMsg.DecodedText,
					"sender_id":    imageMsg.SenderID,
					"sender_name":  senderName,
					"timestamp":    imageMsg.Timestamp,
				}
				jsonBytes, _ := json.Marshal(payload)

				for client := range clients {
					// Send the JSON text message to every client (including sender)
					select {
					case client.Send <- &OutgoingMessage{Type: websocket.TextMessage, Data: jsonBytes}:
						log.Printf("Hub: sent image+text to user=%s in room=%s", client.UserID, imageMsg.RoomID)
					default:
						log.Printf("Hub: send channel blocked for user=%s in room=%s; closing connection", client.UserID, imageMsg.RoomID)
						close(client.Send)
						delete(clients, client)
					}
				}
			}
		}
	}
}
