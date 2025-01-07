package handlers

import (
	"encoding/json"
	"faas-project/internal/models"
	"fmt"
	"net/http"

	//"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Login")
}

func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	var user models.User

	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	user.Password = string(hashedPassword)
	fmt.Println(user)
	w.WriteHeader(http.StatusCreated)

}
