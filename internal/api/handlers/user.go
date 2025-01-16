package handlers

import (
	"encoding/json"
	"faas-project/internal/middleware"
	"faas-project/internal/models"
	"faas-project/internal/repository"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v4"
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
		setResponse(w, http.StatusUnauthorized, "error", "Credenciales inválidas")
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(storedUser.Password), []byte(user.Password))
	if err != nil {
		setResponse(w, http.StatusUnauthorized, "error", "Credenciales inválidas")
		return
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": storedUser.Username,
		"exp": time.Now().Add(24 * time.Hour).Unix(),
	})
	tokenString, err := token.SignedString(middleware.JwtSecret)
	if err != nil {
		setResponse(w, http.StatusInternalServerError, "error", "No se pudo generar el token")
		return
	}

	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "Usuario logueado correctamente",
		"token":   tokenString,
	})
}

func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	var user models.User
	w.Header().Set("Content-Type", "application/json")

	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		setResponse(w, http.StatusBadRequest, "error", err.Error())
		return
	}

	// Verificar si el usuario existe
	existingUser, err := repository.GetUserRepository().GetByUsername(user.Username)
	if err == nil && existingUser.Password != "" {
		setResponse(w, http.StatusConflict, "error", "El usuario ya existe")
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		setResponse(w, http.StatusInternalServerError, "error", "Error al procesar el registro")
		return
	}
	user.Password = string(hashedPassword)
	err = repository.GetUserRepository().CreateUser(user)
	if err != nil {
		setResponse(w, http.StatusInternalServerError, "error", "Error al crear el usuario")
		return
	}

	setResponse(w, http.StatusCreated, "success", "Usuario registrado correctamente")
}

func setResponse(w http.ResponseWriter, status int, statusMessage string, content string) {
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  statusMessage,
		"message": content,
	})
}
