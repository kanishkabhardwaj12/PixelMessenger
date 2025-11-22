package storage

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kanishkabhardwaj12/PixelMessenger/backend/models"
)

var DB *pgxpool.Pool

func InitDB(connString string) {
	var err error
	DB, err = pgxpool.New(context.Background(), connString)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}

	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id UUID PRIMARY KEY,
		username TEXT UNIQUE NOT NULL,
		password_hash TEXT NOT NULL
	);
	CREATE TABLE IF NOT EXISTS rooms (
		id UUID PRIMARY KEY,
		name TEXT NOT NULL,
		owner_id UUID REFERENCES users(id)
	);
	CREATE TABLE IF NOT EXISTS room_members (
		room_id UUID REFERENCES rooms(id) ON DELETE CASCADE,
		user_id UUID REFERENCES users(id) ON DELETE CASCADE,
		PRIMARY KEY (room_id, user_id)
	);
	CREATE TABLE IF NOT EXISTS messages (
		id UUID PRIMARY KEY,
		room_id UUID REFERENCES rooms(id) ON DELETE CASCADE,
		sender_id UUID REFERENCES users(id),
		decoded_text TEXT,
		encoded_image_base64 TEXT,
		original_image_base64 TEXT,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT now()
	);
	`
	_, err = DB.Exec(context.Background(), schema)
	if err != nil {
		log.Fatalf("Failed to create schema: %v\n", err)
	}

	// Migration: Add original_image_base64 column if it doesn't exist
	migration := `
	DO $$ 
	BEGIN
		IF NOT EXISTS (
			SELECT 1 FROM information_schema.columns 
			WHERE table_name='messages' AND column_name='original_image_base64'
		) THEN
			ALTER TABLE messages ADD COLUMN original_image_base64 TEXT;
		END IF;
	END $$;
	`
	_, err = DB.Exec(context.Background(), migration)
	if err != nil {
		log.Printf("Warning: Failed to run migration: %v\n", err)
	}

	fmt.Println("Database initialized successfully.")
}

// --- User Functions ---

func CreateUser(user models.User) error {
	query := "INSERT INTO users (id, username, password_hash) VALUES ($1, $2, $3)"
	_, err := DB.Exec(context.Background(), query, user.ID, user.Username, user.PasswordHash)
	return err
}

func GetUserByUsername(username string) (*models.User, error) {
	var user models.User
	query := "SELECT id, username, password_hash FROM users WHERE username = $1"
	err := DB.QueryRow(context.Background(), query, username).Scan(&user.ID, &user.Username, &user.PasswordHash)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetUserByID returns a user by their UUID id.
func GetUserByID(id string) (*models.User, error) {
	var user models.User
	query := "SELECT id, username, password_hash FROM users WHERE id = $1"
	err := DB.QueryRow(context.Background(), query, id).Scan(&user.ID, &user.Username, &user.PasswordHash)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// --- Room Functions ---

func CreateRoom(room models.Room) error {
	tx, err := DB.Begin(context.Background())
	if err != nil {
		return err
	}
	defer tx.Rollback(context.Background())

	roomQuery := "INSERT INTO rooms (id, name, owner_id) VALUES ($1, $2, $3)"
	_, err = tx.Exec(context.Background(), roomQuery, room.ID, room.Name, room.OwnerID)
	if err != nil {
		return err
	}

	memberQuery := "INSERT INTO room_members (room_id, user_id) VALUES ($1, $2)"
	_, err = tx.Exec(context.Background(), memberQuery, room.ID, room.OwnerID)
	if err != nil {
		return err
	}

	return tx.Commit(context.Background())
}

func AddUserToRoom(roomID, userID string) error {
	query := "INSERT INTO room_members (room_id, user_id) VALUES ($1, $2) ON CONFLICT DO NOTHING"
	_, err := DB.Exec(context.Background(), query, roomID, userID)
	return err
}

func GetRoomByID(roomID string) (*models.Room, error) {
	var room models.Room
	query := "SELECT id, name, owner_id FROM rooms WHERE id = $1"
	err := DB.QueryRow(context.Background(), query, roomID).Scan(&room.ID, &room.Name, &room.OwnerID)
	if err != nil {
		return nil, err
	}
	return &room, nil
}

func GetRoomsForUser(userID string) ([]models.Room, error) {
	var rooms []models.Room
	query := `
		SELECT r.id, r.name, r.owner_id
		FROM rooms r
		JOIN room_members rm ON r.id = rm.room_id
		WHERE rm.user_id = $1
	`
	rows, err := DB.Query(context.Background(), query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var room models.Room
		if err := rows.Scan(&room.ID, &room.Name, &room.OwnerID); err != nil {
			return nil, err
		}
		rooms = append(rooms, room)
	}

	return rooms, nil
}

// Messages
func SaveMessage(msg models.Message) error {
	query := `INSERT INTO messages (id, room_id, sender_id, decoded_text, encoded_image_base64, original_image_base64, created_at) VALUES ($1,$2,$3,$4,$5,$6,$7)`
	_, err := DB.Exec(context.Background(), query, msg.ID, msg.RoomID, msg.SenderID, msg.DecodedText, msg.EncodedImageBase64, msg.OriginalImageBase64, msg.CreatedAt)
	return err
}

func GetMessagesForRoom(roomID string, limit int) ([]models.Message, error) {
	var msgs []models.Message
	if limit <= 0 {
		limit = 100
	}
	query := `SELECT id, room_id, sender_id, decoded_text, encoded_image_base64, COALESCE(original_image_base64, ''), created_at FROM messages WHERE room_id = $1 ORDER BY created_at ASC LIMIT $2`
	rows, err := DB.Query(context.Background(), query, roomID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var m models.Message
		if err := rows.Scan(&m.ID, &m.RoomID, &m.SenderID, &m.DecodedText, &m.EncodedImageBase64, &m.OriginalImageBase64, &m.CreatedAt); err != nil {
			return nil, err
		}
		msgs = append(msgs, m)
	}

	return msgs, nil
}

// DeleteMessage deletes a message by ID
func DeleteMessage(messageID string, userID string) error {
	// Only allow deletion if the user is the sender
	query := `DELETE FROM messages WHERE id = $1 AND sender_id = $2`
	result, err := DB.Exec(context.Background(), query, messageID, userID)
	if err != nil {
		return err
	}
	
	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("message not found or you don't have permission to delete it")
	}
	
	return nil
}
