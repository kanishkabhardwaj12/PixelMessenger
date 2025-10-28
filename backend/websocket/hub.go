package ws

import (
	"bytes"
	"encoding/json"
	"image"
	"log"
	"net/http"

	"github.com/kanishkabhardwaj12/PixelMessenger/backend/steganography"
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
	aiServiceURL := "http://localhost:5000/select-image"

	// 1. Get the best image URL from the AI service
	bestImageURL := getBestImageURL(aiServiceURL)

	// 2. Fetch the image from the URL
	img, err := fetchImage(bestImageURL)
	if err != nil {
		log.Printf("Failed to fetch image: %v", err)
		return // Don't send anything if this fails
	}

	// 3. Encode the message into the image
	encodedImageBuf, err := steganography.Encode(img, message.Message)
	if err != nil {
		log.Printf("Failed to encode steganography: %v", err)
		return // Don't send anything if this fails
	}

	// 4. Send the FINAL encoded image back to the Hub's new channel
	h.roomBroadcast <- &roomBroadcast{
		RoomID:      message.RoomID,
		EncodedData: encodedImageBuf.Bytes(),
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
				for client := range clients {
					select {
					case client.Send <- imageMsg.EncodedData:
					default:
						close(client.Send)
						delete(clients, client)
					}
				}
			}
		}
	}
}
