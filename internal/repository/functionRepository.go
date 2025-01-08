package repository

import (
	"encoding/json"
	"faas-project/internal/message"
	"faas-project/internal/models"
	"os/exec"
	"strings"

	"github.com/nats-io/nats.go"
)

type FunctionRepository interface {
	CreateFunction(function models.Function) error
	GetByName(name string) (models.Function, error)
	DeleteFunction(name string) error
	ExecuteFunction(function models.Function, param string) (string, error)
}

type NATSFunctionRepository struct {
	js nats.JetStreamContext
}

func NewNATSFunctionRepository(js nats.JetStreamContext) *NATSFunctionRepository {
	return &NATSFunctionRepository{js: js}
}

func (r *NATSFunctionRepository) CreateFunction(function models.Function) error {
	kv, err := r.js.KeyValue("functions")
	if err != nil {
		return err
	}

	data, err := json.Marshal(function)
	if err != nil {
		return err
	}

	_, err = kv.Put(function.Name, data)
	return err
}

func (r *NATSFunctionRepository) GetByName(name string) (models.Function, error) {
	kv, err := r.js.KeyValue("functions")
	if err != nil {
		return models.Function{}, err
	}

	entry, err := kv.Get(name)
	if err != nil {
		return models.Function{}, err
	}

	var function models.Function
	err = json.Unmarshal(entry.Value(), &function)
	if err != nil {
		return models.Function{}, err
	}

	return function, nil
}

func (r *NATSFunctionRepository) DeleteFunction(name string) error {
	kv, err := r.js.KeyValue("functions")
	if err != nil {
		return err
	}

	return kv.Delete(name)
}

func (r *NATSFunctionRepository) ExecuteFunction(function models.Function, param string) (string, error) {
	containerName := "function-" + function.Name

	// Eliminar contenedor si existe
	rmCmd := exec.Command("docker", "rm", "-f", containerName)
	rmOutput, rmErr := rmCmd.CombinedOutput()
	if rmErr != nil {
		println("Error al eliminar contenedor:", string(rmOutput))
	}

	// Ejecutar contenedor
	cmd := exec.Command("docker", "run", "--name", containerName,
		"-e", "PARAM="+param,
		"--rm", function.Image,
		"sh", "-c", "echo $PARAM")

	println("Ejecutando comando:", cmd.String()) // Mostrar el comando completo

	output, err := cmd.CombinedOutput()
	if err != nil {
		println("Error de Docker:", err.Error())
		println("Salida de Docker:", string(output))
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}

func GetFunctionRepository() *NATSFunctionRepository {
	js := message.GetJetStream()
	return NewNATSFunctionRepository(js)
} 