package repository

import (
	"encoding/json"
	"faas-project/internal/models"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/client"
	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
)

type PendingFunction struct {
	Function models.Function
	Param    string
}

type ExecutionRequest struct {
	Function    models.Function `json:"function"`
	Param       string          `json:"param"`
	ContainerId string          `json:"containerId"`
}

type FunctionRepository interface {
	CreateFunction(function models.Function) error
	GetByName(name string) (models.Function, error)
	DeleteFunction(name string) error
	AddFunctionToUser(username string, function models.Function) error
	GetFunctionsByUser(username string) ([]models.Function, error)
	GetPendingExecutions() ([]PendingFunction, error)
	Update(function models.Function) error
	GetJS() nats.JetStreamContext
}

type NATSFunctionRepository struct {
	js             nats.JetStreamContext
	docker         *client.Client
	containerLocks sync.Map
	maxConcurrent  int
}

var natsURL = "nats://nats:4222"

func cleanDockerOutput(output string) string {
	if len(output) < 8 {
		return output
	}
	return output[8:]
}
func (r *NATSFunctionRepository) PublishFunction(function models.Function, param string, w http.ResponseWriter) {

	nc, err := nats.Connect(natsURL)
	if err != nil {
		log.Fatal(err)
	}
	defer nc.Close()

	containerId := fmt.Sprintf("faas-%s", uuid.New().String())

	data, err := json.Marshal(ExecutionRequest{
		Function:    function,
		Param:       param,
		ContainerId: containerId,
	})
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{
			"status": "error",
			"msg":    "Error al serializar la solicitud de ejecución" + err.Error(),
		})
		return
	}
	executeSubject := fmt.Sprintf("functions.%s", containerId)
	replySubject := fmt.Sprintf("response.%s", containerId)

	msg := &nats.Msg{
		Subject: executeSubject,
		Data:    data,
		Reply:   replySubject,
	}

	responseChan := make(chan string)

	sub, err := nc.Subscribe(replySubject, func(msg *nats.Msg) {
		log.Printf("Respuesta de ejecución222222: %s", string(msg.Data))
		cleanOuput := cleanDockerOutput(string(msg.Data))
		responseChan <- cleanOuput
	})
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{
			"status": "error",
			"msg":    "Error en subscripción: " + err.Error(),
		})
		return
	}
	defer sub.Unsubscribe()

	nc.PublishMsg(msg)
	select {
	case response := <-responseChan:
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"status": "success",
			"result": response,
		})
	case <-time.After(30 * time.Second):
		w.WriteHeader(http.StatusGatewayTimeout)
		json.NewEncoder(w).Encode(map[string]string{
			"status": "error",
			"msg":    "Timeout esperando respuesta",
		})
	}
}

func NewNATSFunctionRepository(js nats.JetStreamContext) *NATSFunctionRepository {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		panic(err)
	}
	return &NATSFunctionRepository{
		js:            js,
		docker:        cli,
		maxConcurrent: 5,
	}
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
		if err == nats.ErrKeyNotFound {
			return models.Function{}, fmt.Errorf("función %s no encontrada", name)
		}
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

	entry, err := kv.Get(name)
	if err != nil {
		return err
	}

	var function models.Function
	err = json.Unmarshal(entry.Value(), &function)
	if err != nil {
		return err
	}

	err = kv.Delete(name)
	if err != nil {
		return err
	}

	kvUserFunctions, err := r.js.KeyValue("user_functions")
	if err != nil {
		return err
	}

	userFunctionsEntry, err := kvUserFunctions.Get(function.OwnerId)
	if err != nil {
		if err == nats.ErrKeyNotFound {
			return nil
		}
		return err
	}

	var functions []models.Function
	err = json.Unmarshal(userFunctionsEntry.Value(), &functions)
	if err != nil {
		return err
	}

	updatedFunctions := []models.Function{}
	for _, fn := range functions {
		if fn.Name != name {
			updatedFunctions = append(updatedFunctions, fn)
		}
	}

	data, err := json.Marshal(updatedFunctions)
	if err != nil {
		return err
	}

	_, err = kvUserFunctions.Put(function.OwnerId, data)
	if err != nil {
		return err
	}

	return nil
}

func GetFunctionRepository() *NATSFunctionRepository {
	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = "nats://nats:4222"
	}

	nc, err := nats.Connect(natsURL)
	if err != nil {
		log.Printf("Error al conectar con NATS: %v", err)
		return nil
	}

	js, err := nc.JetStream()
	if err != nil {
		log.Printf("Error al obtener el contexto de JetStream: %v", err)
		nc.Close()
		return nil
	}

	_, err = js.AddStream(&nats.StreamConfig{
		Name:     "EXECUTIONS",
		Subjects: []string{"execution.>"},
	})
	if err != nil && !strings.Contains(err.Error(), "stream name already in use") {
		log.Printf("Error al crear el stream de ejecuciones: %v", err)
		nc.Close()
		return nil
	}

	_, err = js.CreateKeyValue(&nats.KeyValueConfig{
		Bucket: "functions",
	})
	if err != nil && err.Error() != "stream name already in use" {
		log.Printf("Error al crear el bucket de funciones: %v", err)
		nc.Close()
		return nil
	}

	_, err = js.CreateKeyValue(&nats.KeyValueConfig{
		Bucket: "user_functions",
	})
	if err != nil && err.Error() != "stream name already in use" {
		log.Printf("Error al crear el bucket de funciones del usuario: %v", err)
		nc.Close()
		return nil
	}

	return NewNATSFunctionRepository(js)
}

func (r *NATSFunctionRepository) AddFunctionToUser(username string, function models.Function) error {
	kv, err := r.js.KeyValue("user_functions")
	if err != nil {
		return err
	}

	entry, err := kv.Get(username)
	var functions []models.Function
	if err == nil {
		err = json.Unmarshal(entry.Value(), &functions)
		if err != nil {
			return err
		}
	} else if err != nats.ErrKeyNotFound {
		return err
	}

	functions = append(functions, function)

	data, err := json.Marshal(functions)
	if err != nil {
		return err
	}

	_, err = kv.Put(username, data)
	return err
}

func (r *NATSFunctionRepository) GetFunctionsByUser(username string) ([]models.Function, error) {
	kv, err := r.js.KeyValue("user_functions")
	if err != nil {
		return nil, err
	}

	entry, err := kv.Get(username)
	if err != nil {
		if err == nats.ErrKeyNotFound {
			return []models.Function{}, nil
		}
		return nil, err
	}

	var functions []models.Function
	err = json.Unmarshal(entry.Value(), &functions)
	if err != nil {
		return nil, err
	}

	return functions, nil
}

func (r *NATSFunctionRepository) GetPendingExecutions() ([]PendingFunction, error) {
	kv, err := r.js.KeyValue("functions")
	if err != nil {
		return nil, err
	}

	entries, err := kv.Keys()
	if err != nil {
		if err == nats.ErrNoKeysFound {
			return []PendingFunction{}, nil
		}
		return nil, err
	}

	var pendingFunctions []PendingFunction
	now := time.Now()

	for _, key := range entries {
		lockKey := fmt.Sprintf("pending_lock_%s", key)
		if _, exists := r.containerLocks.LoadOrStore(lockKey, true); exists {
			continue
		}

		go func(k string) {
			time.Sleep(30 * time.Second)
			r.containerLocks.Delete(fmt.Sprintf("pending_lock_%s", k))
		}(key)

		entry, err := kv.Get(key)
		if err != nil {
			continue
		}

		var function models.Function
		if err := json.Unmarshal(entry.Value(), &function); err != nil {
			continue
		}

		if function.NextExecution.IsZero() || function.NextExecution.Before(now) {
			if function.LastExecution.IsZero() || function.LastExecution.Before(function.NextExecution) {
				pendingFunctions = append(pendingFunctions, PendingFunction{
					Function: function,
					Param:    fmt.Sprintf("test%d", len(pendingFunctions)+1),
				})
			}
		}
	}

	return pendingFunctions, nil
}

func (r *NATSFunctionRepository) Update(function models.Function) error {
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

func (r *NATSFunctionRepository) GetJS() nats.JetStreamContext {
	return r.js
}
