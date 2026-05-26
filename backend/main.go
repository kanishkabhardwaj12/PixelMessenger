package main

import (
	"bufio"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/kanishkabhardwaj12/PixelMessenger/backend/handlers"
	"github.com/kanishkabhardwaj12/PixelMessenger/backend/middleware"
	"github.com/kanishkabhardwaj12/PixelMessenger/backend/storage"
	ws "github.com/kanishkabhardwaj12/PixelMessenger/backend/websocket"
)

// loadEnv loads environment variables from .env file
func loadEnv() {
	envPath := filepath.Join(".", ".env")
	file, err := os.Open(envPath)
	if err != nil {
		log.Println("No .env file found, using system environment variables")
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			// Remove quotes if present
			value = strings.Trim(value, "'\"")
			os.Setenv(key, value)
		}
	}
	log.Println("Environment variables loaded from .env file")
}

func main() {
	// Load environment variables from .env file
	loadEnv()

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
	// --- 3. Create a router and register routes ---
	router := http.NewServeMux()
	// Public routes
	router.HandleFunc("/register", handlers.Register)
	router.HandleFunc("/login", handlers.Login)

	// Protected routes (all require a valid JWT)
	// WebSocket route
	// We call HandleConnections(hub) to *get* the handler function
	router.Handle("/ws", middleware.JwtMiddleware(handlers.HandleConnections(hub))) // <-- 4. PASS THE HUB

	// Room management routes
	router.Handle("/rooms", middleware.JwtMiddleware(http.HandlerFunc(handlers.CreateRoom)))
	router.Handle("/my-rooms", middleware.JwtMiddleware(http.HandlerFunc(handlers.GetUserRooms)))
	router.Handle("/rooms/", middleware.JwtMiddleware(http.HandlerFunc(handlers.RoomSubHandler))) // Catches /rooms/{id}/invite and /rooms/{id}/messages

	// Decode route
	router.Handle("/decode", middleware.JwtMiddleware(http.HandlerFunc(handlers.DecodeImage)))

	// Encode (custom image) route - accepts base64 image + message and broadcasts the
	// resulting encoded PNG into the room. We pass the hub so the handler can publish.
	router.Handle("/encode", middleware.JwtMiddleware(handlers.EncodeImage(hub)))

	// Steganography Analysis route - compares cover and stego images
	router.HandleFunc("/analyze-stego", handlers.AnalyzeSteganography)

	// --- 5. Start Server ---
	// Read desired port from BACKEND_PORT env var (fallback to 8082)
	port := os.Getenv("BACKEND_PORT")
	if port == "" {
		port = "8082"
	}
	addr := ":" + port
	log.Printf("Secure API server starting on %s", addr)
	handler := middleware.CorsMiddleware(router)
	err := http.ListenAndServe(addr, handler)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
