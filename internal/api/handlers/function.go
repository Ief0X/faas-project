package handlers

import (
	"encoding/json"
	"faas-project/internal/models"
	"faas-project/internal/repository"
	"fmt"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v4"
)

func RegisterFunctionHandler(w http.ResponseWriter, r *http.Request) {
	var function models.Function
	w.Header().Set("Content-Type", "application/json")

	err := json.NewDecoder(r.Body).Decode(&function)
	if err != nil {
		setResponse(w, http.StatusBadRequest, "error", err.Error())
		return
	}

	if function.Name == "" || function.Image == "" {
		setResponse(w, http.StatusBadRequest, "error", "Nombre e imagen son requeridos")
		return
	}
	existingFunction, err := repository.GetFunctionRepository().GetFunctionsByUser(function.OwnerId)
	if err == nil && len(existingFunction) > 0 {
		setResponse(w, http.StatusConflict, "error", "Ya existe una función con ese nombre")
		return
	}
	authHeader := r.Header.Get("Authorization")
	userName, err := extractUserFromToken(authHeader)
	if err != nil {
		setResponse(w, http.StatusUnauthorized, "error", "Token inválido")
		return
	}
	if userName != function.OwnerId {
		setResponse(w, http.StatusForbidden, "error", "No tienes permisos para ejecutar esta función")
		return
	}
	err = repository.GetFunctionRepository().CreateFunction(function)
	if err != nil {
		setResponse(w, http.StatusInternalServerError, "error", "Error al registrar la función")
		return
	}
	setResponse(w, http.StatusCreated, "success", "Función registrada exitosamente")
}

func DeleteFunctionHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	functionName := strings.TrimPrefix(r.URL.Path, "/function/")
	if functionName == "" {
		setResponse(w, http.StatusBadRequest, "error", "Nombre de función requerido")
		return
	}
	function, err := repository.GetFunctionRepository().GetFunctionByName(functionName)
	if err != nil {
		setResponse(w, http.StatusNotFound, "error", "Función no encontrada")
		return
	}
	authHeader := r.Header.Get("Authorization")
	userName, err := extractUserFromToken(authHeader)
	if err != nil {
		setResponse(w, http.StatusUnauthorized, "error", "Token inválido")
		return
	}
	if userName != function.OwnerId {
		setResponse(w, http.StatusForbidden, "error", "No tienes permisos para ejecutar esta función")
		return
	}
	err = repository.GetFunctionRepository().DeleteFunction(function)
	if err != nil {
		setResponse(w, http.StatusInternalServerError, "error", "Error al eliminar la función")
		return
	}

	setResponse(w, http.StatusOK, "success", "Función eliminada exitosamente")
}

func ExecuteFunctionHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		setResponse(w, http.StatusMethodNotAllowed, "error", "Método no permitido")
		return
	}

	functionName := strings.TrimPrefix(r.URL.Path, "/function/")
	if functionName == "" {
		setResponse(w, http.StatusBadRequest, "error", "Nombre de función requerido")
		return
	}

	function, err := repository.GetFunctionRepository().GetFunctionByName(functionName)
	if err != nil {
		setResponse(w, http.StatusNotFound, "error", "Función no encontrada")
		return
	}
	authHeader := r.Header.Get("Authorization")
	userName, err := extractUserFromToken(authHeader)
	if err != nil {
		setResponse(w, http.StatusUnauthorized, "error", "Token inválido")
		return
	}
	if userName != function.OwnerId {
		setResponse(w, http.StatusForbidden, "error", "No tienes permisos para ejecutar esta función")
		return
	}

	var param struct {
		Param string `json:"param"`
	}
	err = json.NewDecoder(r.Body).Decode(&param)
	if err != nil {
		setResponse(w, http.StatusBadRequest, "error", "Error al decodificar el parámetro")
		return
	}
	repository.GetFunctionRepository().PublishFunction(function, param.Param, w)
}

func GetFunctionsByUserHandler(w http.ResponseWriter, r *http.Request) {
	username := r.URL.Query().Get("username")
	if username == "" {
		setResponse(w, http.StatusBadRequest, "error", "Nombre de usuario requerido")
		return
	}
	function, err := repository.GetFunctionRepository().GetFunctionsByUser(username)
	if err != nil {
		setResponse(w, http.StatusInternalServerError, "error", "Error al obtener funciones del usuario")
		return
	}
	authHeader := r.Header.Get("Authorization")
	checkedUser, err := extractUserFromToken(authHeader)
	if err != nil {
		setResponse(w, http.StatusUnauthorized, "error", "Token inválido")
		return
	}
	if checkedUser != username {
		setResponse(w, http.StatusForbidden, "error", "Token incorrecto")
		return
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(function)
}

func extractUserFromToken(tokenString string) (string, error) {
	if strings.HasPrefix(tokenString, "Bearer ") {
		tokenString = strings.TrimPrefix(tokenString, "Bearer ")
	} else {
		return "", fmt.Errorf("token inválido")
	}
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("método de firma inesperado: %v", token.Header["alg"])
		}
		return []byte("kohi"), nil
	})
	if err != nil {
		return "", fmt.Errorf("token inválido: %v", err)
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		if username, ok := claims["sub"].(string); ok {
			return username, nil
		}
	}

	return "", fmt.Errorf("token inválido o expirado")
}
