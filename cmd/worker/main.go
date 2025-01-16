package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"faas-project/internal/models"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/nats-io/nats.go"
)

var url = "nats://nats:4222"

func main() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	nc, err := nats.Connect(url)
	if err != nil {
		log.Fatal(err)
	}

	nc.QueueSubscribe(
		"execution.*", "workers",
		func(msg *nats.Msg) {

			hostConfig := &container.HostConfig{
				AutoRemove:  true,
				NetworkMode: container.NetworkMode("faas-project_faas-network"),
			}
			dockerClient, err := client.NewClientWithOpts(client.FromEnv)
			if err != nil {
				log.Printf("Error al crear el cliente de Docker: %v", err)
				return
			}
			log.Printf("Mensaje recibido desde NATS ÑÑÑÑÑ: %s", string(msg.Data))
			var req struct {
				Function models.Function `json:"function"`
				Param    string          `json:"param"`
			}
			if err := json.Unmarshal(msg.Data, &req); err != nil {
				log.Printf("Error al deserializar la solicitud de ejecución: %v", err)
				return
			}

			log.Printf("----------------------------------param: %s", req.Param)

			ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
			defer cancel()
			var randomName = fmt.Sprintf("faas-%d", time.Now().Unix())
			log.Println(req.Function.Image)
			resp, err := dockerClient.ContainerCreate(ctx, &container.Config{
				Image:        req.Function.Image,
				Env:          []string{fmt.Sprintf("PARAM=%s", req.Param)},
				Tty:          false,
				AttachStdout: true,
				AttachStderr: true,
			}, hostConfig, nil, nil, randomName)
			if err != nil {
				log.Printf("Error al crear el contenedor: %v", err)
				return
			}
			err = dockerClient.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{})
			if err != nil {
				log.Printf("Error al iniciar el contenedor: %v", err)
				return
			}
			logOpts := types.ContainerLogsOptions{
				ShowStdout: true,
				ShowStderr: true,
				Follow:     true,
			}
			logReader, err := dockerClient.ContainerLogs(ctx, resp.ID, logOpts)
			if err != nil {
				log.Printf("Error al leer los logs del contenedor: %v", err)
				return
			}
			logCh := make(chan []byte, 1)
			go func() {
				logs, _ := io.ReadAll(logReader)
				logCh <- logs
				close(logCh)
			}()

			statusCh, errCh := dockerClient.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
			select {
			case logs := <-logCh:
				log.Printf("Logs del contenedor: %s", logs)
			case err := <-errCh:
				log.Printf("Error al esperar a que el contenedor termine: %v", err)
				return
			}
			select {
			case status := <-statusCh:
				log.Printf("Estado del contenedor: %d", status.StatusCode)
			case err := <-errCh:
				log.Printf("Error al esperar a que el contenedor termine: %v", err)
				return
			}
			err = dockerClient.ContainerRemove(ctx, resp.ID, types.ContainerRemoveOptions{})
			if err != nil {
				log.Printf("Error al eliminar el contenedor: %v", err)
				return
			}

			nc.Publish(msg.Reply, []byte("Ejecución exitosa"))

		})
	<-sigChan
}
