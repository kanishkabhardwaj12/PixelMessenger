package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/kanishkabhardwaj12/PixelMessenger/backend/models"
	"github.com/kanishkabhardwaj12/PixelMessenger/backend/storage"
)

func CreateRoom(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad Request: Invalid JSON", http.StatusBadRequest)
		return
	}
	if req.Name == "" {
		http.Error(w, "Bad Request: Room name is required", http.StatusBadRequest)
		return
	}

	userID := r.Context().Value("userID").(string)

	newRoom := models.Room{
		ID:      uuid.New().String(),
		Name:    req.Name,
		OwnerID: userID,
	}

	if err := storage.CreateRoom(newRoom); err != nil {
		http.Error(w, "Failed to create room", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(newRoom)

}

// RoomSubHandler handles room sub-resources such as inviting users and
// retrieving messages. It is mounted at /rooms/ and inspects the suffix
// to determine the intended action (e.g. /rooms/{id}/invite or
// /rooms/{id}/messages).
func RoomSubHandler(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/rooms/")

	// messages endpoint: GET /rooms/{id}/messages
	if strings.HasSuffix(r.URL.Path, "/messages") && r.Method == http.MethodGet {
		roomID := strings.TrimSuffix(path, "/messages")
		msgs, err := storage.GetMessagesForRoom(roomID, 100)
		if err != nil {
			http.Error(w, "Failed to retrieve messages", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(msgs)
		return
	}

	// delete message endpoint: DELETE /rooms/{roomId}/messages/{messageId}
	if strings.Contains(r.URL.Path, "/messages/") && r.Method == http.MethodDelete {
		parts := strings.Split(path, "/messages/")
		if len(parts) != 2 {
			http.Error(w, "Bad Request: Invalid URL format", http.StatusBadRequest)
			return
		}
		messageID := parts[1]
		userID := r.Context().Value("userID").(string)

		err := storage.DeleteMessage(messageID, userID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": "Message deleted successfully"})
		return
	}

	// invite endpoint: POST /rooms/{id}/invite
	if strings.HasSuffix(r.URL.Path, "/invite") && r.Method == http.MethodPost {
		var req struct {
			Username string `json:"username"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Bad Request: Invalid JSON", http.StatusBadRequest)
			return
		}

		roomID := strings.TrimSuffix(path, "/invite")
		inviterID := r.Context().Value("userID").(string)

		room, err := storage.GetRoomByID(roomID)
		if err != nil {
			http.Error(w, "Room not found", http.StatusBadRequest)
			return
		}

		if room.OwnerID != inviterID {
			http.Error(w, "Forbidden: Only the room owner can invite users", http.StatusForbidden)
			return
		}

		invitee, err := storage.GetUserByUsername(req.Username)
		if err != nil {
			http.Error(w, "User to invite not found", http.StatusNotFound)
			return
		}

		if err := storage.AddUserToRoom(room.ID, invitee.ID); err != nil {
			http.Error(w, "Failed to invite user to room", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": "User invited successfully"})
		return
	}

	http.Error(w, "Not Found", http.StatusNotFound)
}

func GetUserRooms(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("userID").(string)

	userRooms, err := storage.GetRoomsForUser(userID)
	if err != nil {
		http.Error(w, "Failed to retrieve rooms", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(userRooms)

}
