package ws

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/png"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/kanishkabhardwaj12/PixelMessenger/backend/models"
	"github.com/kanishkabhardwaj12/PixelMessenger/backend/steganography"
	"github.com/kanishkabhardwaj12/PixelMessenger/backend/storage"
)

// BroadcastMessage is a TEXT message from a client to the Hub.
type BroadcastMessage struct {
	RoomID      string
	UserID      string
	Message     []byte // This is the raw text message
	MessageType string // "text" or "delete"
}

// roomBroadcast is an IMAGE message from a worker to the Hub.
// This is a new internal message type.
type roomBroadcast struct {
	RoomID       string
	MessageID    string // UUID of the message in database
	EncodedData  []byte // This is the final encoded PNG
	OriginalData []byte // This is the original image (before encoding)
	SenderID     string // user who triggered this broadcast (do not resend to them)
	DecodedText  string // The decoded message extracted from the payload
	Timestamp    string // RFC3339 timestamp
}

type Hub struct {
	Rooms      map[string]map[*Client]bool
	Broadcast  chan *BroadcastMessage // Receives TEXT from clients
	Register   chan *Client
	Unregister chan *Client

	// This is the new channel. Workers send the FINAL image here.
	roomBroadcast chan *roomBroadcast
	
	// Image cache to avoid re-downloading
	imageCache map[string]image.Image
	cacheMutex sync.RWMutex // Protects imageCache
}

func NewHub() *Hub {
	hub := &Hub{
		Broadcast:     make(chan *BroadcastMessage),
		Register:      make(chan *Client),
		Unregister:    make(chan *Client),
		Rooms:         make(map[string]map[*Client]bool),
		roomBroadcast: make(chan *roomBroadcast), // Initialize the new channel
		imageCache:    make(map[string]image.Image), // Initialize image cache
	}
	
	// Pre-warm cache with first 5 images in background
	go func() {
		log.Println("Pre-warming image cache...")
		for i := 0; i < 5 && i < len(sampleImageURLs); i++ {
			url := sampleImageURLs[i]
			img, err := fetchImage(url)
			if err != nil {
				log.Printf("Failed to pre-cache image %s: %v", url, err)
				continue
			}
			hub.cacheMutex.Lock()
			hub.imageCache[url] = img
			hub.cacheMutex.Unlock()
			log.Printf("Pre-cached image %d/%d", i+1, 5)
		}
		log.Println("Image cache pre-warming complete")
	}()
	
	return hub
}

// PublishEncodedImage allows external callers to submit a ready-encoded PNG
// which the hub will broadcast into the given room. This is used by handlers
// that accept user-uploaded images and want the hub to deliver them.
func (h *Hub) PublishEncodedImage(roomID string, messageID string, data []byte, senderID string, decodedText string, originalData []byte) {
	h.roomBroadcast <- &roomBroadcast{
		RoomID:       roomID,
		MessageID:    messageID,
		EncodedData:  data,
		OriginalData: originalData,
		SenderID:     senderID,
		DecodedText:  decodedText,
		Timestamp:    time.Now().UTC().Format(time.RFC3339),
	}
}

// --- Helper functions (getBestImageURL, fetchImage) are unchanged ---

var sampleImageURLs = []string{
	// Nature & Landscapes
	"https://images.unsplash.com/photo-1500964757637-c85e8a162699?ixlib=rb-4.0.3&q=80&fm=jpg&w=1080",
	"https://images.unsplash.com/photo-1472214103451-9374bd1c798e?ixlib=rb-4.0.3&q=80&fm=jpg&w=1080",
	"https://images.unsplash.com/photo-1470071459604-3b5ec3a7fe05?ixlib=rb-4.0.3&q=80&fm=jpg&w=1080",
	"https://images.unsplash.com/photo-1447752875215-b2761acb3c5d?ixlib=rb-4.0.3&q=80&fm=jpg&w=1080",
	"https://images.unsplash.com/photo-1469474968028-56623f02e42e?ixlib=rb-4.0.3&q=80&fm=jpg&w=1080",
	"https://images.unsplash.com/photo-1506905925346-21bda4d32df4?ixlib=rb-4.0.3&q=80&fm=jpg&w=1080",
	"https://images.unsplash.com/photo-1441974231531-c6227db76b6e?ixlib=rb-4.0.3&q=80&fm=jpg&w=1080",
	"https://images.unsplash.com/photo-1426604966848-d7adac402bff?ixlib=rb-4.0.3&q=80&fm=jpg&w=1080",
	
	// Mountains & Sky
	"https://images.unsplash.com/photo-1506905925346-21bda4d32df4?ixlib=rb-4.0.3&q=80&fm=jpg&w=1080",
	"https://images.unsplash.com/photo-1519904981063-b0cf448d479e?ixlib=rb-4.0.3&q=80&fm=jpg&w=1080",
	"https://images.unsplash.com/photo-1464822759023-fed622ff2c3b?ixlib=rb-4.0.3&q=80&fm=jpg&w=1080",
	
	// Ocean & Water
	"https://images.unsplash.com/photo-1505142468610-359e7d316be0?ixlib=rb-4.0.3&q=80&fm=jpg&w=1080",
	"https://images.unsplash.com/photo-1439066615861-d1af74d74000?ixlib=rb-4.0.3&q=80&fm=jpg&w=1080",
	"https://images.unsplash.com/photo-1507525428034-b723cf961d3e?ixlib=rb-4.0.3&q=80&fm=jpg&w=1080",
	
	// Flowers & Plants
	"https://images.unsplash.com/photo-1490750967868-88aa4486c946?ixlib=rb-4.0.3&q=80&fm=jpg&w=1080",
	"https://images.unsplash.com/photo-1508962914676-134849a727f0?ixlib=rb-4.0.3&q=80&fm=jpg&w=1080",
	"https://images.unsplash.com/photo-1502082553048-f009c37129b9?ixlib=rb-4.0.3&q=80&fm=jpg&w=1080",
	
	// Abstract & Patterns
	"https://images.unsplash.com/photo-1557672172-298e090bd0f1?ixlib=rb-4.0.3&q=80&fm=jpg&w=1080",
	"https://images.unsplash.com/photo-1579546929518-9e396f3cc809?ixlib=rb-4.0.3&q=80&fm=jpg&w=1080",
	"https://images.unsplash.com/photo-1558591710-4b4a1ae0f04d?ixlib=rb-4.0.3&q=80&fm=jpg&w=1080",
	
	// Cityscapes & Architecture
	"https://images.unsplash.com/photo-1480714378408-67cf0d13bc1b?ixlib=rb-4.0.3&q=80&fm=jpg&w=1080",
	"https://images.unsplash.com/photo-1514565131-fce0801e5785?ixlib=rb-4.0.3&q=80&fm=jpg&w=1080",
	"https://images.unsplash.com/photo-1449824913935-59a10b8d2000?ixlib=rb-4.0.3&q=80&fm=jpg&w=1080",
	
	// Sunset & Sunrise
	"https://images.unsplash.com/photo-1495616811223-4d98c6e9c869?ixlib=rb-4.0.3&q=80&fm=jpg&w=1080",
	"https://images.unsplash.com/photo-1506443432602-ac2fcd6f54e0?ixlib=rb-4.0.3&q=80&fm=jpg&w=1080",
	"https://images.unsplash.com/photo-1475924156734-496f6cac6ec1?ixlib=rb-4.0.3&q=80&fm=jpg&w=1080",
}

func getBestImageURL(aiServiceURL string, message string) string {
	payload, _ := json.Marshal(map[string]interface{}{
		"image_urls": sampleImageURLs,
		"message":    message, // Send message to AI service for context-aware selection
	})
	reqBody := bytes.NewBuffer(payload)

	resp, err := http.Post(aiServiceURL, "application/json", reqBody)
	if err != nil {
		log.Printf("AI service call failed: %v, using hash-based fallback", err)
		// Fallback: use simple hash of message to pick an image
		hash := 0
		for _, ch := range message {
			hash = (hash*31 + int(ch)) % len(sampleImageURLs)
		}
		if hash < 0 {
			hash = -hash
		}
		selectedURL := sampleImageURLs[hash%len(sampleImageURLs)]
		log.Printf("Selected image (hash fallback): %s", selectedURL)
		return selectedURL
	}
	defer resp.Body.Close()

	var result map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Printf("Failed to decode AI response: %v, using hash-based fallback", err)
		// Fallback: use simple hash of message to pick an image
		hash := 0
		for _, ch := range message {
			hash = (hash*31 + int(ch)) % len(sampleImageURLs)
		}
		if hash < 0 {
			hash = -hash
		}
		selectedURL := sampleImageURLs[hash%len(sampleImageURLs)]
		log.Printf("Selected image (hash fallback): %s", selectedURL)
		return selectedURL
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
	// Fast hash-based image selection (skip slow AI service call and URL download)
	messageText := string(message.Message)
	
	// Use simple hash to pick an image URL
	hash := 0
	for _, ch := range messageText {
		hash = (hash*31 + int(ch)) % len(sampleImageURLs)
	}
	if hash < 0 {
		hash = -hash
	}
	bestImageURL := sampleImageURLs[hash%len(sampleImageURLs)]
	
	// 2. Check cache first, then fetch if needed
	var img image.Image
	var err error
	
	// Try to read from cache
	h.cacheMutex.RLock()
	cachedImg, found := h.imageCache[bestImageURL]
	h.cacheMutex.RUnlock()
	
	if found {
		img = cachedImg
		log.Printf("Using cached image for URL: %s", bestImageURL)
	} else {
		img, err = fetchImage(bestImageURL)
		if err != nil {
			log.Printf("Failed to fetch image: %v", err)
			return // Don't send anything if this fails
		}
		// Cache the image for future use
		h.cacheMutex.Lock()
		h.imageCache[bestImageURL] = img
		h.cacheMutex.Unlock()
		log.Printf("Cached new image from URL: %s", bestImageURL)
	}

	// 3. Save the original image as PNG bytes
	originalImageBuf := new(bytes.Buffer)
	if err := png.Encode(originalImageBuf, img); err != nil {
		log.Printf("Failed to encode original image as PNG: %v", err)
		return
	}
	originalImageBytes := originalImageBuf.Bytes()

	// 4. Encode the message into the image
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

	// 5. Create message ID and save to database
	decodedText := string(message.Message)
	msgID := generateMessageID()
	msg := models.Message{
		ID:                  msgID,
		RoomID:              message.RoomID,
		SenderID:            message.UserID,
		DecodedText:         decodedText,
		EncodedImageBase64:  base64.StdEncoding.EncodeToString(encodedImageBuf.Bytes()),
		OriginalImageBase64: base64.StdEncoding.EncodeToString(originalImageBytes),
		CreatedAt:           time.Now().UTC(),
	}
	if err := storage.SaveMessage(msg); err != nil {
		log.Printf("Failed to save text message to DB: %v", err)
	}

	// 6. Send BOTH the original and encoded images back to the Hub's new channel
	// We also include the decoded text and a timestamp so the hub can broadcast
	// a JSON message containing both images (base64) and the decoded text.
	log.Printf("Worker: encoded image for room=%s sender=%s size=%d bytes", message.RoomID, message.UserID, encodedImageBuf.Len())
	h.roomBroadcast <- &roomBroadcast{
		RoomID:       message.RoomID,
		MessageID:    msgID,
		EncodedData:  encodedImageBuf.Bytes(),
		OriginalData: originalImageBytes,
		SenderID:     message.UserID,
		DecodedText:  decodedText,
		Timestamp:    time.Now().UTC().Format(time.RFC3339),
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
			// Handle different message types
			if message.MessageType == "delete" {
				// Parse delete message
				var deleteMsg map[string]interface{}
				if err := json.Unmarshal(message.Message, &deleteMsg); err == nil {
					if msgID, ok := deleteMsg["messageId"].(string); ok {
						// Broadcast delete to all clients in the room
						if clients, ok := h.Rooms[message.RoomID]; ok {
							payload := map[string]interface{}{
								"type":      "delete",
								"messageId": msgID,
							}
							jsonBytes, _ := json.Marshal(payload)
							
							for client := range clients {
								select {
								case client.Send <- &OutgoingMessage{Type: websocket.TextMessage, Data: jsonBytes}:
									log.Printf("Hub: sent delete notification to user=%s in room=%s", client.UserID, message.RoomID)
								default:
									log.Printf("Hub: send channel blocked for user=%s", client.UserID)
								}
							}
						}
					}
				}
			} else {
				// A client sent a text message for steganography encoding.
				// DO NOT do the work here.
				// Spin off a new goroutine to handle it.
				log.Printf("Broadcast received for room %s, delegating to worker", message.RoomID)
				go h.processSteganography(message)
			}

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

				// Prepare a JSON payload containing BOTH original and encoded images + decoded text + metadata
				// Note: Message is already saved to database by the worker (processSteganography)
				payload := map[string]interface{}{
					"type":                  "image",
					"message_id":            imageMsg.MessageID,
					"image_base64":          base64.StdEncoding.EncodeToString(imageMsg.EncodedData),
					"original_image_base64": base64.StdEncoding.EncodeToString(imageMsg.OriginalData),
					"decoded_text":          imageMsg.DecodedText,
					"sender_id":             imageMsg.SenderID,
					"sender_name":           senderName,
					"timestamp":             imageMsg.Timestamp,
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

// generateMessageID generates a unique ID for a message
func generateMessageID() string {
	return uuid.New().String()
}
