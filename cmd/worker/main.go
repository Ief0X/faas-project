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

type channelWriter struct {
	channel chan []byte
}

func (cw channelWriter) Write(p []byte) (n int, err error) {
	cw.channel <- p
	return len(p), nil
}

func main() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	nc, err := nats.Connect(url)

	if err != nil {
		log.Fatal(err)
	}

	nc.QueueSubscribe(
		"functions.*", "workers",
		func(msg *nats.Msg) {

			dockerClient, err := client.NewClientWithOpts(client.FromEnv)
			if err != nil {
				log.Printf("Error al crear el cliente de Docker: %v", err)
				return
			}

			var req struct {
				Function    models.Function `json:"function"`
				Param       string          `json:"param"`
				ContainerId string          `json:"containerId"`
			}
			if err := json.Unmarshal(msg.Data, &req); err != nil {
				log.Printf("Error al deserializar la solicitud de ejecución: %v", err)
				return
			}

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			hostConfig := &container.HostConfig{
				AutoRemove:  true,
				NetworkMode: container.NetworkMode("faas-project_faas-network"),
			}
			reader, err := dockerClient.ImagePull(ctx, req.Function.Image, types.ImagePullOptions{})
			if err != nil {
				log.Printf("No se ha encontrado la imagen en docker.io: %v", err)
				return
			}

			_, err = io.Copy(os.Stdout, reader)
			if err != nil {
				log.Printf("Error al copiar la salida del pull: %v", err)
				return
			}
			resp, err := dockerClient.ContainerCreate(ctx, &container.Config{
				Image:        req.Function.Image,
				Env:          []string{fmt.Sprintf("PARAM=%s", req.Param)},
				Tty:          false,
				AttachStdout: true,
				AttachStderr: true,
			}, hostConfig, nil, nil, req.ContainerId)
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
				return
			}

			logCh := make(chan []byte, 1)
			go func() {
				defer close(logCh)
				_, err := io.Copy(channelWriter{logCh}, logReader)
				if err != nil {
					return
				}
			}()

			statusCh, errCh := dockerClient.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)

			select {
			case logs := <-logCh:
				log.Printf("Logs del contenedor: %s", logs)
				nc.Publish(msg.Reply, logs)
			case err := <-errCh:
				log.Printf("Error al esperar a que el contenedor termine: %v", err)
				nc.Publish(msg.Reply, []byte(err.Error()))

			case status := <-statusCh:
				log.Printf("Estado del contenedor: %d", status.StatusCode)
			}

		})
	<-sigChan

}
