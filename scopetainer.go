package scopetainer

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"
)

type Image interface {
	Name() string
	String() string
}

type image struct {
	registry, name, tag string
}

type ContainerConfig struct {
	Image                Image
	TCPPorts             []int
	UDPPorts             []int
	EnvironmentVariables map[string]string
}

func ImageFrom(registry, name, tag string) (Image, error) {
	if name == "" {
		return nil, fmt.Errorf("must have name provided")
	}
	if tag == "" {
		tag = "latest"
	}
	return image{
		registry: registry,
		name:     name,
		tag:      tag,
	}, nil
}

func (i image) Name() string {
	return i.name
}

func (i image) String() string {
	main := strings.Trim(strings.Join([]string{i.registry, i.name}, "/"), "/")
	return main + ":" + i.tag
}

type Container struct {
	PortBinding map[int]int
}

type HealthCheck func(context.Context, Container) error

func AsTCPHealthChecks(ports []int, timeout time.Duration) []HealthCheck {
	healthChecks := make([]HealthCheck, 0, len(ports))
	for _, port := range ports {
		healthChecks = append(healthChecks, TCPHealthCheck(port, timeout))
	}
	return healthChecks
}

func TCPHealthCheck(port int, timeout time.Duration) HealthCheck {
	return func(ctx context.Context, c Container) error {
		hostPort, wasBound := c.PortBinding[port]
		if !wasBound {
			return fmt.Errorf("%d was not bound to the host", port)
		}

		ticker := time.NewTicker(time.Millisecond * 250)
		defer ticker.Stop()
		success := make(chan struct{})
		attemptPort := func() {
			connection, err := net.DialTimeout("tcp", fmt.Sprintf(`127.0.0.1:%d`, hostPort), time.Second*3)
			if err != nil {
				return
			}
			if connection != nil {
				success <- struct{}{}
			}
		}

		defer ticker.Stop()
		healthCheckCtx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		for {
			select {
			case <-ticker.C:
				go attemptPort()
			case <-success:
				return nil
			case <-healthCheckCtx.Done():
				return fmt.Errorf("port %d did not come alive after %s", port, timeout)
			}
		}
	}
}

type Broker interface {
	Start(ctx context.Context, containerConfig ContainerConfig, healthChecks ...HealthCheck) (container Container, cleanup func(ctx context.Context) error, err error)
}
