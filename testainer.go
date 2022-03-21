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
	// The dockerhub registry, this is used by default in the cli but must
	// be done explicitly when programmatically used with the daemon.
	DockerHubLibraryRegistry = `registry.hub.docker.com/library`
)

// Congfiguration to start the image with
type Config struct {
	Registry string
	Image    string
	Tag      string
	Port     int
	Env      map[string]string
}

// Generic information about the created container
type ContainerDetails struct {
	Port int
}

type Testainer[ConfigT, ContainerT any] interface {
	Use(ctx context.Context, config ConfigT, callback CallbackFunc[ContainerT]) error
	Run(ctx context.Context, config ConfigT) (*ContainerT, CleanupFunc, error)
}

type testainer[ConfigT, ContainerT any] struct {
	docker         *dkr.Client
	mapConfigFunc  common.MapFunc[ConfigT, Config]
	mapDetailsFunc common.MapFunc[ContainerDetails, ContainerT]
}

type CleanupFunc func() error

type CallbackFunc[ContainerT any] func(ctx context.Context, details *ContainerT) error

type dockerCreationConfig struct {
	hostPort    int
	guestConfig container.Config
	hostConfig  container.HostConfig
}

func New[ConfigT, ContainerT any](toConfig common.MapFunc[ConfigT, Config], fromContainerDetails common.MapFunc[ContainerDetails, ContainerT]) (Testainer[ConfigT, ContainerT], error) {
	docker, err := dkr.NewClientWithOpts(dkr.FromEnv)
	if err != nil {
		return nil, fmt.Errorf("could not create docker client: %v", err)
	}
	return testainer[ConfigT, ContainerT]{
		docker:         docker,
		mapConfigFunc:  toConfig,
		mapDetailsFunc: fromContainerDetails,
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
	config := t.mapConfigFunc(configT)
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
	containerT := t.mapDetailsFunc(ContainerDetails{
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

// Takes a registry, image, and tag and returns a fully quallified docker
// image name. `image` is requred, but registry and tag are optional.
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

// Converts key value pairs into a slice of strings where each item
// takes the `key=value` format docker wants.
func mapAsDockerEnv(m map[string]string) []string {
	dockerEnvSlice := make([]string, 0, len(m))
	for k, v := range m {
		dockerEnvSlice = append(dockerEnvSlice, fmt.Sprintf("%s=%s", k, v))
	}
	return dockerEnvSlice
}

// Will wait for the TCP port on `127.0.0.1` to respond to connections.
// This will check indefinitely until the context is closed. This will
// return `false` if the context closes before a connection succeeds.
// This will return `true` if the port is successfully connected. If the
// context is not properly setup to timeout, this will wait indefinitely.
func checkTCPPort(ctx context.Context, port int) bool {
	successChannel := make(chan struct{}, 1)
	defer close(successChannel)
	isDone := false
	go func() {
		for !isDone {
			connection, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), time.Second*3)
			if err != nil {
				time.Sleep(time.Millisecond * 500)
				continue
			}
			if connection != nil {
				defer connection.Close()
				successChannel <- struct{}{}
				isDone = true
			}
		}
	}()

	select {
	case <-successChannel:
		return true
	case <-ctx.Done():
		isDone = true
		return false
	}
}
