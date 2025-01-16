package handlers

import (
	"encoding/json"
	"faas-project/internal/models"
	"faas-project/internal/repository"
	"net/http"
	"strings"
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

	existingFunction, err := repository.GetFunctionRepository().GetByName(function.Name)
	if err == nil && existingFunction.ID != "" {
		setResponse(w, http.StatusConflict, "error", "Ya existe una función con ese nombre")
		return
	}

	err = repository.GetFunctionRepository().CreateFunction(function)
	if err != nil {
		setResponse(w, http.StatusInternalServerError, "error", "Error al crear la función")
		return
	}

	err = repository.GetFunctionRepository().AddFunctionToUser(function.OwnerId, function)
	if err != nil {
		setResponse(w, http.StatusInternalServerError, "error", "Error al asociar la función al usuario")
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

	_, err := repository.GetFunctionRepository().GetByName(functionName)
	if err != nil {
		setResponse(w, http.StatusNotFound, "error", "Función no encontrada")
		return
	}

	err = repository.GetFunctionRepository().DeleteFunction(functionName)
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

	function, err := repository.GetFunctionRepository().GetByName(functionName)
	if err != nil {
		setResponse(w, http.StatusNotFound, "error", "Función no encontrada")
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
	msg, err := repository.GetFunctionRepository().PublishFunction(function, param.Param)
	if err != nil {
		setResponse(w, http.StatusInternalServerError, "error", "Error al publicar la función")
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "success",
		"msg":    msg.Subject,
	})
}

func GetFunctionsByUserHandler(w http.ResponseWriter, r *http.Request) {
	username := r.URL.Query().Get("username")
	if username == "" {
		setResponse(w, http.StatusBadRequest, "error", "Nombre de usuario requerido")
		return
	}

	functions, err := repository.GetFunctionRepository().GetFunctionsByUser(username)
	if err != nil {
		setResponse(w, http.StatusInternalServerError, "error", "Error al obtener funciones del usuario")
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(functions)
}
