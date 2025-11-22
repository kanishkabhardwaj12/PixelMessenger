package models

import "time"

// Message represents a saved chat message (possibly with an encoded image)
type Message struct {
    ID                 string    `json:"id"`
    RoomID             string    `json:"room_id"`
    SenderID           string    `json:"sender_id"`
    DecodedText        string    `json:"decoded_text"`
    EncodedImageBase64 string    `json:"encoded_image_base64"`
    OriginalImageBase64 string   `json:"original_image_base64,omitempty"` // NEW: Store original image
    CreatedAt          time.Time `json:"created_at"`
}
