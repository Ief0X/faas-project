package repository

import (
	"context"
	"encoding/json"
	"faas-project/internal/message"
	"faas-project/internal/models"
	"io"
	"strings"

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
}

type NATSFunctionRepository struct {
	js     nats.JetStreamContext
	docker *client.Client
}

func NewNATSFunctionRepository(js nats.JetStreamContext) *NATSFunctionRepository {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		panic(err)
	}
	return &NATSFunctionRepository{
		js:     js,
		docker: cli,
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

	return kv.Delete(name)
}

func (r *NATSFunctionRepository) ExecuteFunction(function models.Function, param string) (string, error) {
	ctx := context.Background()
	containerName := "function-" + function.Name

	r.docker.ContainerRemove(ctx, containerName, types.ContainerRemoveOptions{
		Force: true,
	})

	config := &container.Config{
		Image: function.Image,
		Cmd:   []string{"sh", "-c", "echo $PARAM"},
		Env:   []string{"PARAM=" + param},
	}

	resp, err := r.docker.ContainerCreate(ctx, config, nil, nil, nil, containerName)
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

	out, err := r.docker.ContainerLogs(ctx, resp.ID, types.ContainerLogsOptions{ShowStdout: true})
	if err != nil {
		return "", err
	}
	defer out.Close()

	logs, err := io.ReadAll(out)
	if err != nil {
		return "", err
	}

	output := strings.TrimSpace(string(logs[8:]))
	return output, nil
}

func GetFunctionRepository() *NATSFunctionRepository {
	js := message.GetJetStream()
	return NewNATSFunctionRepository(js)
} 