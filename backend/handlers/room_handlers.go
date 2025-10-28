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

func InviteToRoom(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad Request: Invalid JSON", http.StatusBadRequest)
		return
	}

	roomID := strings.TrimPrefix(r.URL.Path, "/rooms/")
	roomID = strings.TrimSuffix(roomID, "/invite")

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
