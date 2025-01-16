package repository

import (
	"context"
	"encoding/json"
	"errors"
	"faas-project/internal/models"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/nats-io/nats.go"
)

type PendingFunction struct {
	Function models.Function
	Param    string
}

type FunctionRepository interface {
	CreateFunction(function models.Function) error
	GetByName(name string) (models.Function, error)
	DeleteFunction(name string) error
	ExecuteFunction(function models.Function, param string) (string, error)
	AddFunctionToUser(username string, function models.Function) error
	GetFunctionsByUser(username string) ([]models.Function, error)
	GetPendingExecutions() ([]PendingFunction, error)
	Update(function models.Function) error
	GetJS() nats.JetStreamContext
}

type NATSFunctionRepository struct {
	js               nats.JetStreamContext
	docker           *client.Client
	activeExecutions sync.Map
	containerLocks   sync.Map
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

func (r *NATSFunctionRepository) ExecuteFunction(function models.Function, param string) (string, error) {
	if os.Getenv("IS_WORKER") == "true" {
		return r.executeInWorker(function, param)
	}

	type executionRequest struct {
		Function models.Function `json:"function"`
		Param    string         `json:"param"`
	}

	req := executionRequest{
		Function: function,
		Param:    param,
	}

	data, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("error serializando la solicitud: %v", err)
	}

	_, err = r.js.PublishMsg(&nats.Msg{
		Subject: fmt.Sprintf("execution.%s", function.Name),
		Data:    data,
	}, nats.MsgId(function.Name))
	if err != nil {
		return "", fmt.Errorf("error publicando la solicitud de ejecución: %v", err)
	}

	sub, err := r.js.PullSubscribe(fmt.Sprintf("execution.%s.response", function.Name), "")
	if err != nil {
		return "", fmt.Errorf("error suscribiendo a la respuesta: %v", err)
	}
	defer sub.Unsubscribe()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return "", fmt.Errorf("tiempo de espera agotado para la respuesta de ejecución")
		default:
			msgs, err := sub.Fetch(1, nats.Context(ctx))
			if err != nil {
				if err == context.DeadlineExceeded {
					return "", fmt.Errorf("tiempo de espera agotado para la respuesta de ejecución")
				}
				continue
			}
			if len(msgs) > 0 {
				return string(msgs[0].Data), nil
			}
		}
	}
}

func (r *NATSFunctionRepository) executeInWorker(function models.Function, param string) (string, error) {
	log.Printf("Iniciando ejecución de función %s con parámetro: %s", function.Name, param)

	if !r.canExecute(function.Name) {
		log.Printf("Se alcanzó el máximo de ejecuciones concurrentes para la función %s", function.Name)
		return "", errors.New("se alcanzó el máximo de ejecuciones concurrentes")
	}
	defer r.removeExecution(function.Name)

	originalFunction, err := r.GetByName(function.Name)
	if err != nil {
		log.Printf("Función %s no encontrada: %v", function.Name, err)
		return "", fmt.Errorf("función no encontrada: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	containerName := fmt.Sprintf("function-%s-%d-%s", 
		function.Name, 
		time.Now().UnixNano(),
		strings.ReplaceAll(param, " ", "_"))

	containers, err := r.docker.ContainerList(ctx, types.ContainerListOptions{All: true})
	if err == nil {
		for _, cont := range containers {
			if strings.HasPrefix(cont.Names[0], "/function-"+function.Name) {
				_ = r.docker.ContainerRemove(ctx, cont.ID, types.ContainerRemoveOptions{
					Force: true,
				})
			}
		}
	}

	containerLock := fmt.Sprintf("container_lock_%s", containerName)
	if _, exists := r.containerLocks.LoadOrStore(containerLock, true); exists {
		return "", fmt.Errorf("el contenedor ya está siendo gestionado")
	}
	defer r.containerLocks.Delete(containerLock)

	containers, err = r.docker.ContainerList(ctx, types.ContainerListOptions{
		All: true,
	})
	if err != nil {
		log.Printf("Error al listar contenedores: %v", err)
		return "", fmt.Errorf("error al listar contenedores: %v", err)
	}

	for _, cont := range containers {
		for _, name := range cont.Names {
			if strings.HasPrefix(name[1:], fmt.Sprintf("function-%s-", function.Name)) {
				log.Printf("Eliminando contenedor antiguo %s", cont.ID)
				err := r.docker.ContainerRemove(ctx, cont.ID, types.ContainerRemoveOptions{
					Force:         true,
					RemoveVolumes: true,
				})
				if err != nil {
					log.Printf("Error al eliminar contenedor %s: %v", cont.ID, err)
				}
				time.Sleep(2 * time.Second)
			}
		}
	}

	containers, err = r.docker.ContainerList(ctx, types.ContainerListOptions{
		All: true,
	})
	if err != nil {
		return "", fmt.Errorf("error al verificar contenedores: %v", err)
	}
	for _, cont := range containers {
		for _, name := range cont.Names {
			if name == "/"+containerName {
				return "", fmt.Errorf("el contenedor todavía existe después de la limpieza")
			}
		}
	}

	log.Printf("Creando contenedor para la función %s con la imagen %s", function.Name, function.Image)

	config := &container.Config{
		Image:        function.Image,
		Env:          []string{fmt.Sprintf("PARAM=%s", strings.TrimSpace(param))},
		Tty:          false,
		AttachStdout: true,
		AttachStderr: true,
	}

	hostConfig := &container.HostConfig{
		AutoRemove:  false,
		NetworkMode: container.NetworkMode("faas-project_faas-network"),
	}

	resp, err := r.docker.ContainerCreate(ctx, config, hostConfig, nil, nil, containerName)
	if err != nil {
		return "", fmt.Errorf("error al crear el contenedor: %v", err)
	}

	for i := 0; i < 30; i++ {
		_, _, err := r.docker.ImageInspectWithRaw(ctx, function.Image)
		if err == nil {
			break
		}
		log.Printf("Esperando a que la imagen %s esté disponible... intento %d", function.Image, i+1)
		time.Sleep(2 * time.Second)
	}

	if err := r.docker.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		log.Printf("Error starting container for function %s: %v", function.Name, err)
		return "", fmt.Errorf("error al iniciar el contenedor: %v", err)
	}
	log.Printf("Contenedor iniciado correctamente para la función %s", function.Name)

	time.Sleep(2 * time.Second)

	logOpts := types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
	}
	logReader, err := r.docker.ContainerLogs(ctx, resp.ID, logOpts)
	if err != nil {
		return "", fmt.Errorf("error al obtener los logs del contenedor: %v", err)
	}
	defer logReader.Close()

	logCh := make(chan []byte, 1)
	go func() {
		logs, _ := io.ReadAll(logReader)
		logCh <- logs
		close(logCh)
	}()

	var result string
	var execError error

	statusCh, errCh := r.docker.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			execError = fmt.Errorf("error al esperar el contenedor: %v", err)
		}
	case status := <-statusCh:
		if status.StatusCode != 0 {
			select {
			case logs := <-logCh:
				errLogs := string(logs)
				log.Printf("Logs del contenedor para %s: %s", function.Name, errLogs)
				execError = fmt.Errorf("el contenedor salió con el estado %d: %s", status.StatusCode, errLogs)
			case <-time.After(5 * time.Second):
				execError = fmt.Errorf("el contenedor salió con el estado %d, timeout al obtener los logs", status.StatusCode)
			}
		} else {
			select {
			case logs := <-logCh:
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
				result = strings.Join(cleanLines, "\n")
			case <-time.After(5 * time.Second):
				execError = fmt.Errorf("timeout al obtener los logs del contenedor")
			}
		}
	case <-ctx.Done():
		log.Printf("Timeout ejecutando la función %s, matando el contenedor...", function.Name)
		_ = r.docker.ContainerKill(context.Background(), resp.ID, "SIGKILL")
		execError = fmt.Errorf("timeout al esperar la ejecución del contenedor")
	}

	cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cleanupCancel()
	err = r.docker.ContainerRemove(cleanupCtx, resp.ID, types.ContainerRemoveOptions{
		Force:         true,
		RemoveVolumes: true,
	})
	if err != nil {
		log.Printf("Error al eliminar el contenedor %s: %v", resp.ID, err)
	}
	log.Printf("Limpieza del contenedor %s completada", resp.ID)

	if execError != nil {
		originalFunction.LastExecution = time.Now()
		originalFunction.LastResult = execError.Error()
		originalFunction.NextExecution = time.Now().Add(1 * time.Minute)
		if updateErr := r.Update(originalFunction); updateErr != nil {
			log.Printf("Error al actualizar el estado de la función después de un error: %v", updateErr)
		}
		return "", execError
	}

	originalFunction.LastExecution = time.Now()
	originalFunction.LastResult = result
	originalFunction.NextExecution = time.Now().Add(5 * time.Minute)
	if err := r.Update(originalFunction); err != nil {
		log.Printf("Error al actualizar el estado de la función después de un éxito: %v", err)
	}

	return result, nil
}

func (r *NATSFunctionRepository) canExecute(functionName string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	count := 0
	r.activeExecutions.Range(func(key, value interface{}) bool {
		if keyStr, ok := key.(string); ok {
			if !strings.HasPrefix(keyStr, "container_lock_") && 
			   !strings.HasPrefix(keyStr, "lock_") && 
			   !strings.HasPrefix(keyStr, "pending_lock_") {
				if timestamp, ok := value.(time.Time); ok {
					if time.Since(timestamp) < 30*time.Second {
						count++
					} else {
						r.activeExecutions.Delete(keyStr)
					}
				}
			}
		}
		return true
	})

	if count >= r.maxConcurrent {
		return false
	}

	r.activeExecutions.Store(functionName, time.Now())
	return true
}

func (r *NATSFunctionRepository) removeExecution(functionName string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if _, exists := r.activeExecutions.Load(functionName); exists {
		r.activeExecutions.Delete(functionName)
	}
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
