package main

import (
	"log"
	"net/http"
	"os" // Import the 'os' package

	"github.com/kanishkabhardwaj12/PixelMessenger/backend/handlers"
	"github.com/kanishkabhardwaj12/PixelMessenger/backend/middleware"
	"github.com/kanishkabhardwaj12/PixelMessenger/backend/storage"
	ws "github.com/kanishkabhardwaj12/PixelMessenger/backend/websocket"
)

func main() {
	// --- 1. Initialize Database ---
	connString := os.Getenv("DATABASE_URL")
	if connString == "" {
		log.Fatal("DATABASE_URL environment variable is not set")
	}
	storage.InitDB(connString)
	log.Println("Database initialized.")

	// --- 2. Initialize WebSocket Hub ---
	hub := ws.NewHub() // <-- 2. CREATE THE HUB
	go hub.Run()       // <-- 3. RUN THE HUB IN THE BACKGROUND
	log.Println("WebSocket Hub started.")

	// --- 3. Public routes ---
	http.HandleFunc("/register", handlers.Register)
	http.HandleFunc("/login", handlers.Login)

	// --- 4. Protected routes (all require a valid JWT) ---

	// WebSocket route
	// We call HandleConnections(hub) to *get* the handler function
	http.Handle("/ws", middleware.JwtMiddleware(handlers.HandleConnections(hub))) // <-- 4. PASS THE HUB

	// Room management routes
	http.Handle("/rooms", middleware.JwtMiddleware(http.HandlerFunc(handlers.CreateRoom)))
	http.Handle("/my-rooms", middleware.JwtMiddleware(http.HandlerFunc(handlers.GetUserRooms)))
	http.Handle("/rooms/", middleware.JwtMiddleware(http.HandlerFunc(handlers.InviteToRoom))) // Catches /rooms/{id}/invite

	// Decode route
	http.Handle("/decode", middleware.JwtMiddleware(http.HandlerFunc(handlers.DecodeImage)))

	// --- 5. Start Server ---
	log.Println("Secure API server started on :8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
