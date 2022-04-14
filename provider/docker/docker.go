package docker

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"strconv"
	"time"

	"github.com/docker/docker/api/types"
	ctnr "github.com/docker/docker/api/types/container"
	dkr "github.com/docker/docker/client"
	"github.com/docker/docker/errdefs"
	"github.com/docker/go-connections/nat"
	"github.com/google/uuid"
	"github.com/paul-nelson-baker/go-scopetainer"
	"github.com/phayes/freeport"
	"golang.org/x/sync/errgroup"
)

const DockerHubLibrary = `registry.hub.docker.com/library`

type dockerBroker struct {
	client *dkr.Client
}

func New() (scopetainer.Broker, error) {
	docker, err := dkr.NewClientWithOpts(dkr.FromEnv)
	if err != nil {
		return nil, fmt.Errorf("could not create docker client: %v", err)
	}
	return &dockerBroker{
		client: docker,
	}, nil
}

func (broker *dockerBroker) Start(ctx context.Context, containerConfig scopetainer.ContainerConfig, healthChecks ...scopetainer.HealthCheck) (scopetainer.Container, func(ctx context.Context) error, error) {
	hostByGuestPorts, portMap, err := createPortMap(containerConfig.TCPPorts, containerConfig.UDPPorts)
	if err != nil {
		return scopetainer.Container{}, nil, fmt.Errorf("could not create port map: %w", err)
	}
	hostConfig := ctnr.HostConfig{
		PortBindings: portMap,
	}
	guestConfig := ctnr.Config{
		Image: containerConfig.Image.String(),
		Env:   mapAsDockerEnv(containerConfig.EnvironmentVariables),
	}
	if imagePullReader, err := broker.client.ImagePull(ctx, containerConfig.Image.String(), types.ImagePullOptions{}); err != nil {
		return scopetainer.Container{}, nil, fmt.Errorf("could not start to pull image: %w", err)
	} else {
		defer imagePullReader.Close()
		if _, err := io.Copy(ioutil.Discard, imagePullReader); err != nil {
			return scopetainer.Container{}, nil, fmt.Errorf("could not pull image: %w", err)
		}
	}
	dockerContainer, err := broker.client.ContainerCreate(ctx, &guestConfig, &hostConfig, nil, nil, containerConfig.Image.Name()+"_"+uuid.NewString())
	if err != nil {
		return scopetainer.Container{}, nil, fmt.Errorf("could not create container: %w", err)
	}
	cleanupCallback := broker.createCleanupCallback(dockerContainer.ID)
	if err := broker.client.ContainerStart(ctx, dockerContainer.ID, types.ContainerStartOptions{}); err != nil {
		cleanupCallback(ctx)
		return scopetainer.Container{}, nil, fmt.Errorf("container did not start: %w", err)
	}
	container := scopetainer.Container{
		PortBinding: hostByGuestPorts,
	}
	if err := awaitHealthy(ctx, container, healthChecks...); err != nil {
		cleanupCallback(ctx)
		return scopetainer.Container{}, nil, fmt.Errorf("container did not become healthy: %w", err)
	}
	return container, cleanupCallback, nil
}

func (broker dockerBroker) createCleanupCallback(containerID string) func(context.Context) error {
	return func(ctx context.Context) error {
		var err error
		for {
			select {
			case <-ctx.Done():
				if err != nil {
					return err
				}
				return ctx.Err()
			default:
				if err = broker.cleanupContainer(ctx, containerID); err == nil {
					return nil
				}
			}
		}
	}
}

func (broker dockerBroker) cleanupContainer(ctx context.Context, containerID string) error {
	stat, err := broker.client.ContainerStats(ctx, containerID, false)
	if errdefs.IsNotFound(err) {
		return nil
	} else if err != nil {
		return fmt.Errorf("could not get container status: %w", err)
	}
	defer stat.Body.Close()
	stopTimeout := time.Duration(time.Second * 5)
	err = broker.client.ContainerStop(ctx, containerID, &stopTimeout)
	if err != nil {
		return fmt.Errorf("could not stop container: %w", err)
	}
	err = broker.client.ContainerRemove(ctx, containerID, types.ContainerRemoveOptions{
		RemoveVolumes: true,
		Force:         true,
	})
	if err != nil {
		return fmt.Errorf("could not remove container: %w", err)
	}
	return nil
}

func createPortMap(tcpPorts, udpPorts []int) (map[int]int, nat.PortMap, error) {
	hostIP := `0.0.0.0`
	hostByGuestPorts := make(map[int]int)
	portMap := nat.PortMap{}
	for _, tcpPort := range tcpPorts {
		ensureHostPortAllocated(tcpPort, hostByGuestPorts)
		portMapKey, err := nat.NewPort("tcp", strconv.Itoa(tcpPort))
		if err != nil {
			return nil, nil, fmt.Errorf("could not use port %d for tcp: %w", tcpPort, err)
		}
		portMap[portMapKey] = append(portMap[portMapKey], nat.PortBinding{
			HostIP:   hostIP,
			HostPort: strconv.Itoa(hostByGuestPorts[tcpPort]),
		})
	}
	for _, udpPort := range udpPorts {
		ensureHostPortAllocated(udpPort, hostByGuestPorts)
		portMapKey, err := nat.NewPort("tcp", strconv.Itoa(udpPort))
		if err != nil {
			return nil, nil, fmt.Errorf("could not use port %d for tcp: %w", udpPort, err)
		}
		portMap[portMapKey] = append(portMap[portMapKey], nat.PortBinding{
			HostIP:   hostIP,
			HostPort: strconv.Itoa(hostByGuestPorts[udpPort]),
		})
	}
	return hostByGuestPorts, portMap, nil
}

func ensureHostPortAllocated(port int, hostPortByGuestPort map[int]int) error {
	if _, alreadyAllocated := hostPortByGuestPort[port]; !alreadyAllocated {
		hostPort, err := freeport.GetFreePort()
		if err != nil {
			return fmt.Errorf("could not allocate host port: %w", err)
		}
		hostPortByGuestPort[port] = hostPort
	}
	return nil
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

func awaitHealthy(ctx context.Context, container scopetainer.Container, healthChecks ...scopetainer.HealthCheck) error {
	g, ctx := errgroup.WithContext(ctx)
	for i := range healthChecks {
		currentHealthCheck := healthChecks[i]
		g.Go(func() error {
			return currentHealthCheck(ctx, container)
		})
	}
	return g.Wait()
}
