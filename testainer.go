package testainer

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	dkr "github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/paul-nelson-baker/go-testainer/internal/common"
	"github.com/phayes/freeport"
)

const (
	DockerHubLibraryRegistry = `registry.hub.docker.com/library`
)

type Config struct {
	Registry string
	Image    string
	Tag      string
	Port     int
	Env      map[string]string
}

type ContainerDetails struct {
	Port int
}

type Testainer[ConfigT, ContainerT any] interface {
	Use(ctx context.Context, config ConfigT, callback CallbackFunc[ContainerT]) error
	Run(ctx context.Context, config ConfigT) (*ContainerT, CleanupFunc, error)
}

type testainer[ConfigT, ContainerT any] struct {
	docker          *dkr.Client
	toConfigFunc    common.ConvertFunc[ConfigT, Config]
	fromDetailsFunc common.ConvertFunc[ContainerDetails, ContainerT]
}

type CleanupFunc func() error

type CallbackFunc[ContainerT any] func(ctx context.Context, details *ContainerT) error

type dockerCreationConfig struct {
	hostPort    int
	guestConfig container.Config
	hostConfig  container.HostConfig
}

func New[ConfigT, ContainerT any](toConfig common.ConvertFunc[ConfigT, Config], fromContainerDetails common.ConvertFunc[ContainerDetails, ContainerT]) (Testainer[ConfigT, ContainerT], error) {
	docker, err := dkr.NewClientWithOpts(dkr.FromEnv)
	if err != nil {
		return nil, fmt.Errorf("could not create docker client: %v", err)
	}
	return testainer[ConfigT, ContainerT]{
		docker:          docker,
		toConfigFunc:    toConfig,
		fromDetailsFunc: fromContainerDetails,
	}, nil
}

func (t testainer[ConfigT, ContainerT]) Use(ctx context.Context, config ConfigT, callback CallbackFunc[ContainerT]) error {
	containerDetails, cleanupFunc, err := t.Run(ctx, config)
	if err != nil {
		return err
	}
	defer cleanupFunc()
	return callback(ctx, containerDetails)
}

func (t testainer[ConfigT, ContainerT]) Run(ctx context.Context, configT ConfigT) (*ContainerT, CleanupFunc, error) {
	config := t.toConfigFunc(configT)
	dockerCreationConfig, err := createContainerConfig(config)
	if err != nil {
		return nil, nil, fmt.Errorf("could not create docker host/container config: %w", err)
	}
	fullyQualfiedImageName, err := formatImageString(config.Registry, config.Image, config.Tag)
	if err != nil {
		return nil, nil, fmt.Errorf("could not determine fully qualified image name: %w", err)
	}
	imagePullReader, err := t.docker.ImagePull(ctx, fullyQualfiedImageName, types.ImagePullOptions{})
	if err != nil {
		return nil, nil, fmt.Errorf("could not pull image: %w", err)
	}
	defer imagePullReader.Close()
	if _, err := io.Copy(os.Stderr, imagePullReader); err != nil {
		return nil, nil, fmt.Errorf("problem occurred while pulling image: %w", err)
	}
	container, err := t.docker.ContainerCreate(
		ctx, &dockerCreationConfig.guestConfig, &dockerCreationConfig.hostConfig, nil, nil,
		fmt.Sprintf("%s_%d", config.Image, time.Now().Unix()),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("could not create container: %w", err)
	}
	dockerCleanup := t.createCleanupCallback(container.ID)
	if err := t.docker.ContainerStart(ctx, container.ID, types.ContainerStartOptions{}); err != nil {
		defer dockerCleanup()
		return nil, nil, fmt.Errorf("could not start container: %w", err)
	}
	checkTcpCtx, checkTcpCtxCancel := context.WithTimeout(ctx, time.Duration(60*time.Second))
	defer checkTcpCtxCancel()
	if portOpen := checkTCPPort(checkTcpCtx, dockerCreationConfig.hostPort); !portOpen {
		defer dockerCleanup()
		return nil, nil, fmt.Errorf("port never opened: %d", dockerCreationConfig.hostPort)
	}
	containerT := t.fromDetailsFunc(ContainerDetails{
		Port: dockerCreationConfig.hostPort,
	})
	return &containerT, dockerCleanup, nil
}

func (t testainer[ConfigT, ContainerT]) createCleanupCallback(containerID string) func() error {
	var cleanupOnce sync.Once
	return func() error {
		var err error = nil
		cleanupOnce.Do(func() {
			ctx, ctxCancel := context.WithTimeout(context.Background(), time.Duration(time.Second*60))
			defer ctxCancel()
			timeout := time.Duration(time.Second * 5)
			err = t.docker.ContainerStop(ctx, containerID, &timeout)
			if err != nil {
				err = fmt.Errorf("could stop container: %w", err)
				return
			}
			err = t.docker.ContainerRemove(ctx, containerID, types.ContainerRemoveOptions{
				RemoveVolumes: true,
				Force:         true,
			})
			if err != nil {
				err = fmt.Errorf("could not remove stopped container: %w", err)
				return
			}
		})
		return err
	}
}

func createContainerConfig(c Config) (dockerCreationConfig, error) {
	image, err := formatImageString(c.Registry, c.Image, c.Tag)
	if err != nil {
		return dockerCreationConfig{}, fmt.Errorf("couldn't format docker image name: %w", err)
	}
	if c.Port <= 0 {
		return dockerCreationConfig{}, fmt.Errorf("port must be a non-negative integer, but was %d", c.Port)
	}
	containerPort, err := nat.NewPort("tcp", strconv.Itoa(c.Port))
	if err != nil {
		return dockerCreationConfig{}, fmt.Errorf("couldn't create cointainer port: %w", err)
	}
	hostPort, err := freeport.GetFreePort()
	if err != nil {
		return dockerCreationConfig{}, fmt.Errorf("couldn't find free host port: %w", err)
	}

	guestConfig := container.Config{
		Image: image,
		Env:   mapAsDockerEnv(c.Env),
	}
	hostConfig := container.HostConfig{
		PortBindings: nat.PortMap{
			containerPort: []nat.PortBinding{
				{
					HostIP:   "0.0.0.0", // Should this be bound to the loopback address instead?
					HostPort: strconv.Itoa(hostPort),
				},
			},
		},
	}
	return dockerCreationConfig{
		hostPort:    hostPort,
		guestConfig: guestConfig,
		hostConfig:  hostConfig,
	}, nil
}

func formatImageString(registry, image, tag string) (string, error) {
	result := ""
	if registry != "" {
		result += registry + "/"
	}
	if image == "" {
		return "", errors.New("image cannot be empty")
	}
	result += image
	if tag == "" {
		result += ":latest"
	} else {
		result += ":" + tag
	}
	return result, nil
}

func mapAsDockerEnv(m map[string]string) []string {
	dockerEnvSlice := make([]string, 0, len(m))
	for k, v := range m {
		dockerEnvSlice = append(dockerEnvSlice, fmt.Sprintf("%s=%s", k, v))
	}
	return dockerEnvSlice
}

func checkTCPPort(ctx context.Context, port int) bool {
	successChannel := make(chan struct{}, 1)
	defer close(successChannel)
	failed := false
	go func() {
		for !failed {
			connection, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), time.Second*3)
			if err != nil {
				time.Sleep(time.Second * 3)
				continue
			}
			if connection != nil {
				defer connection.Close()
				successChannel <- struct{}{}
				return
			}
		}
	}()

	select {
	case <-successChannel:
		return true
	case <-ctx.Done():
		failed = true
		return false
	}
}
