package testainer

import (
	"context"
	"errors"
	"fmt"
	"sync"

	dkr "github.com/docker/docker/client"
)

var (
	globalClientOnce sync.Once
	globalClient     Testainer = nil
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

type Testainer interface {
	Use(ctx context.Context, config Config, callback CallbackFunc) error
	Run(ctx context.Context, config Config) (ContainerDetails, CleanupFunc, error)
}

type testainer struct {
	docker *dkr.Client
}

func NewTestainer() (Testainer, error) {
	docker, err := dkr.NewClientWithOpts(dkr.FromEnv)
	if err != nil {
		return nil, fmt.Errorf("could not create docker client: %v", err)
	}
	return testainer{
		docker: docker,
	}, nil
}

func (t testainer) Use(ctx context.Context, config Config, callback CallbackFunc) error {
	containerDetails, cleanupFunc, err := t.Run(ctx, config)
	if err != nil {
		return err
	}
	defer cleanupFunc()
	return callback(ctx, containerDetails)
}

func (t testainer) Run(ctx context.Context, config Config) (ContainerDetails, CleanupFunc, error) {
	return ContainerDetails{}, nil, errors.New("not implemented")
}

type CleanupFunc func() error

type CallbackFunc func(ctx context.Context, containerDetails ContainerDetails) error

func Use(ctx context.Context, config Config, callback CallbackFunc) error {
	globalClientInit()
	return globalClient.Use(ctx, config, callback)
}

func Run(ctx context.Context, config Config) (ContainerDetails, CleanupFunc, error) {
	globalClientInit()
	return globalClient.Run(ctx, config)
}

func globalClientInit() {
	globalClientOnce.Do(func() {
		var err error
		globalClient, err = NewTestainer()
		if err != nil {
			panic("could not initialize testainer client: " + err.Error())
		}
	})
}
