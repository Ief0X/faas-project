package repository

import (
	"context"
	"encoding/json"
	"errors"
	"faas-project/internal/message"
	"faas-project/internal/models"
	"io"
	"strings"
	"sync"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/nats-io/nats.go"
)

type FunctionRepository interface {
	CreateFunction(function models.Function) error
	GetByName(name string) (models.Function, error)
	DeleteFunction(name string) error
	ExecuteFunction(function models.Function, param string) (string, error)
	AddFunctionToUser(username string, function models.Function) error
	GetFunctionsByUser(username string) ([]models.Function, error)
}

type NATSFunctionRepository struct {
	js               nats.JetStreamContext
	docker           *client.Client
	activeExecutions sync.Map
	maxConcurrent    int
	mu               sync.Mutex
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

func (r *NATSFunctionRepository) ExecuteFunction(function models.Function, param string) (string, error) {
	if !r.canExecute(function.Name) {
		return "", errors.New("maximum concurrent executions reached")
	}
	defer r.removeExecution(function.Name)

	ctx := context.Background()
	containerName := "function-" + function.Name

	r.docker.ContainerRemove(ctx, containerName, types.ContainerRemoveOptions{
		Force: true,
	})

	config := &container.Config{
		Image:        function.Image,
		Env:          []string{"PARAM=" + param},
		Tty:          false,
		AttachStdout: true,
		AttachStderr: true,
	}

	hostConfig := &container.HostConfig{
		AutoRemove: true,
	}

	resp, err := r.docker.ContainerCreate(ctx, config, hostConfig, nil, nil, containerName)
	if err != nil {
		return "", err
	}

	if err := r.docker.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		return "", err
	}

	statusCh, errCh := r.docker.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			return "", err
		}
	case <-statusCh:
	}

	logReader, err := r.docker.ContainerLogs(ctx, resp.ID, types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     false,
	})
	if err != nil {
		return "", err
	}
	defer logReader.Close()

	logs, err := io.ReadAll(logReader)
	if err != nil {
		return "", err
	}

	r.docker.ContainerRemove(ctx, resp.ID, types.ContainerRemoveOptions{
		Force: true,
	})

	output := string(logs)
	output = strings.TrimSpace(output)

	var cleanLines []string
	for _, line := range strings.Split(output, "\n") {
		if len(line) > 8 {
			cleanLine := line[8:]
			if !strings.Contains(cleanLine, "Procesando parámetro") &&
				!strings.Contains(cleanLine, "JSON parseado") &&
				!strings.Contains(cleanLine, "No es JSON") {
				cleanLines = append(cleanLines, cleanLine)
			}
		}
	}

	return strings.Join(cleanLines, "\n"), nil
}

func (r *NATSFunctionRepository) canExecute(functionName string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	count := 0
	r.activeExecutions.Range(func(_, _ interface{}) bool {
		count++
		return true
	})

	if count >= r.maxConcurrent {
		return false
	}

	r.activeExecutions.Store(functionName, struct{}{})
	return true
}

func (r *NATSFunctionRepository) removeExecution(functionName string) {
	r.activeExecutions.Delete(functionName)
}

func GetFunctionRepository() *NATSFunctionRepository {
	js := message.GetJetStream()
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
