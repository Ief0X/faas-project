package handlers

import (
	"encoding/json"
	"faas-project/internal/models"
	"faas-project/internal/repository"
	"net/http"

	//"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	var user models.User
	w.Header().Set("Content-Type", "application/json")

	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		setResponse(w, http.StatusBadRequest, "error", err.Error())
		return
	}

	storedUser, err := repository.GetUserRepository().GetByUsername(user.Username)
	if err != nil {
		setResponse(w, http.StatusUnauthorized, "error", "Invalid credentials")
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(storedUser.Password), []byte(user.Password))
	if err != nil {
		setResponse(w, http.StatusUnauthorized, "error", "Invalid credentials")
		return
	}

	setResponse(w, http.StatusOK, "success", "User logged in successfully")
}

func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	var user models.User
	w.Header().Set("Content-Type", "application/json")

	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		setResponse(w, http.StatusBadRequest, "error", err.Error())
		return
	}

	user, _ = repository.GetUserRepository().GetByUsername(user.Username)
	if user.Password != "" {
		setResponse(w, http.StatusConflict, "error", "User already exists")
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		setResponse(w, http.StatusInternalServerError, "error", "Error processing registration")
		return
	}
	user.Password = string(hashedPassword)
	err = repository.GetUserRepository().CreateUser(user)
	if err != nil {
		setResponse(w, http.StatusInternalServerError, "error", "Error creating user")
		return
	}

	setResponse(w, http.StatusCreated, "success", "User registered successfully")
}

func setResponse(w http.ResponseWriter, status int, statusMessage string, content string) {
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  statusMessage,
		"message": content,
	})
}
