package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/kanishkabhardwaj12/PixelMessenger/backend/auth"
	"github.com/kanishkabhardwaj12/PixelMessenger/backend/models"
	"github.com/kanishkabhardwaj12/PixelMessenger/backend/storage"
)

type Credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func Register(w http.ResponseWriter, r *http.Request) {
	var creds Credentials

	if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	hashedPassword, err := auth.HashPassword(creds.Password)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	_, err = storage.GetUserByUsername(creds.Username)
	if err == nil {
		http.Error(w, "Username already exists", http.StatusConflict)
		return
	}

	newUser := models.User{
		ID:           uuid.New().String(),
		Username:     creds.Username,
		PasswordHash: hashedPassword,
	}

	if err := storage.CreateUser(newUser); err != nil {
		http.Error(w, "Failed to create user", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"message": "User created successfully"})

}

func Login(w http.ResponseWriter, r *http.Request) {
	var creds Credentials

	if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	user, err := storage.GetUserByUsername(creds.Username)
	if err != nil {
		http.Error(w, "Invalid username", http.StatusUnauthorized)
		return
	}

	if !auth.CheckPasswordHash(creds.Password, user.PasswordHash) {
		http.Error(w, "Invalid password", http.StatusUnauthorized)
		return
	}

	tokenString, err := auth.GenerateJWT(user.Username, user.ID)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"token": tokenString})
}
