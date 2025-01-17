package repository

import (
	"encoding/json"
	"faas-project/internal/models"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

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
type NatsFunctionRepository struct {
	conn *nats.Conn
	js   nats.JetStreamContext
}

var NatsConnection *nats.Conn
var jsGlobal nats.JetStreamContext

var natsURL = "nats://nats:4222"
var REQUEST_TTL, _ = strconv.Atoi(os.Getenv("REQUEST_TTL"))

func cleanDockerOutput(output string) string {
	if len(output) < 8 {
		return strings.TrimSpace(output)
	}
	return strings.TrimSpace(output[8:])
}
func (*NatsFunctionRepository) PublishFunction(function models.Function, param string, w http.ResponseWriter) {

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
	case <-time.After(time.Duration(REQUEST_TTL) * time.Second):
		w.WriteHeader(http.StatusGatewayTimeout)
		json.NewEncoder(w).Encode(map[string]string{
			"status": "error",
			"msg":    "Timeout esperando respuesta",
		})
	}
}

func GetFunctionRepository() *NatsFunctionRepository {
	if NatsConnection == nil || jsGlobal == nil {
		return initFunctionRepository()
	} else {
		return &NatsFunctionRepository{
			conn: NatsConnection,
			js:   jsGlobal,
		}
	}
}
func initFunctionRepository() *NatsFunctionRepository {
	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = "nats://nats:4222"
	}

	nc, err := nats.Connect(natsURL)
	if err != nil {
		log.Printf("Error al conectar con NATS: %v", err)
		return nil
	}
	NatsConnection = nc
	js, err := nc.JetStream()
	if err != nil {
		log.Printf("Error al obtener el contexto de JetStream: %v", err)
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
	NatsConnection = nc
	jsGlobal = js

	return &NatsFunctionRepository{
		conn: nc,
		js:   js,
	}
}
func (r *NatsFunctionRepository) CreateFunction(function models.Function) error {
	kv, err := r.js.KeyValue("user_functions")
	if err != nil {
		return err
	}
	userFunctionsEntry, err := kv.Get(function.OwnerId)
	var functions []models.Function

	if err != nil {
		if err == nats.ErrKeyNotFound {
			functions = []models.Function{}
		} else {
			return err
		}
	} else {
		err = json.Unmarshal(userFunctionsEntry.Value(), &functions)
		if err != nil {
			return err
		}
	}

	functions = append(functions, function)
	data, err := json.Marshal(functions)
	if err != nil {
		return err
	}

	_, err = kv.Put(function.OwnerId, data)
	return err
}

func (r *NatsFunctionRepository) GetFunctionByName(name string) (models.Function, error) {
	kv, err := r.js.KeyValue("user_functions")
	if err != nil {
		return models.Function{}, err
	}

	allUsers, err := kv.Keys()
	if err != nil {
		return models.Function{}, err
	}

	for _, userId := range allUsers {
		entry, err := kv.Get(userId)
		if err != nil {
			continue
		}

		var functions []models.Function
		err = json.Unmarshal(entry.Value(), &functions)
		if err != nil {
			continue
		}

		for _, function := range functions {
			if function.Name == name {
				return function, nil
			}
		}
	}

	return models.Function{}, fmt.Errorf("function not found")
}

func (r *NatsFunctionRepository) DeleteFunction(function models.Function) error {
	kv, err := r.js.KeyValue("user_functions")
	if err != nil {
		return err
	}

	entry, err := kv.Get(function.OwnerId)
	if err != nil {
		return err
	}

	var functions []models.Function
	err = json.Unmarshal(entry.Value(), &functions)
	if err != nil {
		return err
	}

	var updatedFunctions []models.Function
	for _, f := range functions {
		if f.Name != function.Name {
			updatedFunctions = append(updatedFunctions, f)
		}
	}

	data, err := json.Marshal(updatedFunctions)
	if err != nil {
		return err
	}

	_, err = kv.Put(function.OwnerId, data)
	return err
}

func (r *NatsFunctionRepository) GetFunctionsByUser(ownerId string) ([]models.Function, error) {
	kv, err := r.js.KeyValue("user_functions")
	if err != nil {
		return nil, err
	}

	entry, err := kv.Get(ownerId)
	if err != nil {
		if err == nats.ErrKeyNotFound {
			return []models.Function{}, nil
		}
		return nil, err
	}
	var functions []models.Function
	err = json.Unmarshal(entry.Value(), &functions)
	if err != nil {
		var singleFunction models.Function
		err = json.Unmarshal(entry.Value(), &singleFunction)
		if err != nil {
			return nil, err
		}
		functions = []models.Function{singleFunction}
	}
	return functions, nil
}

func (r *NatsFunctionRepository) Update(function models.Function) error {
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

func (r *NatsFunctionRepository) GetJS() nats.JetStreamContext {
	return r.js
}
