package handlers

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/kanishkabhardwaj12/PixelMessenger/backend/storage"
	ws "github.com/kanishkabhardwaj12/PixelMessenger/backend/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// In production, you'd check against your actual frontend domain.
		// For development, true is fine.
		return true
	},
}

func HandleConnections(hub *ws.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// --- 1. Authentication (from middleware) ---
		// We get the UserID from the context, which was set by JwtMiddleware.
		userID := r.Context().Value("userID").(string)
		if userID == "" {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		// --- 2. Get Room ID ---
		// The client will specify the room by a query param, e.g., /ws?room=...
		roomID := r.URL.Query().Get("room")
		if roomID == "" {
			http.Error(w, "Bad Request: 'room' query parameter is required", http.StatusBadRequest)
			return
		}

		// --- 3. Authorization (Check Database) ---
		// We must check if this authenticated user is actually a member of this room.
		rooms, err := storage.GetRoomsForUser(userID)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		isMember := false
		for _, room := range rooms {
			if room.ID == roomID {
				isMember = true
				break
			}
		}

		if !isMember {
			http.Error(w, "Forbidden: You are not a member of this room", http.StatusForbidden)
			return
		}

		// --- 4. Upgrade Connection ---
		// This is correct, 'upgrader' is from 'gorilla/websocket'
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println(err)
			return
		}

		// --- 5. Create and Register Client ---
		// Use the 'ws' alias for our internal Client type
		client := &ws.Client{
			Hub:    hub,
			Conn:   conn, // This is a *websocket.Conn from gorilla, which is correct
			Send:   make(chan []byte, 256),
			RoomID: roomID,
			UserID: userID,
		}
		client.Hub.Register <- client

		go client.WritePump()
		go client.ReadPump()
	}
}
