package docker

import (
	"context"
	"testing"
	"time"

	"github.com/paul-nelson-baker/go-scopetainer"
	"github.com/stretchr/testify/assert"
)

func TestRun(t *testing.T) {
	testcases := []struct {
		image    string
		tags     []string
		tcpPorts []int
		udpPorts []int
		env      map[string]string
	}{
		{
			image: "nginx",
			tags: []string{
				"1.21",
				"stable-alpine",
				"mainline-alpine",
				"alpine",
			},
			tcpPorts: []int{80},
		},
		{
			image: "postgres",
			tags: []string{
				"alpine",
				"alpine3.15",
				"12.0",
				"13.0",
				"14.0",
			},
			tcpPorts: []int{5432},
			env: map[string]string{
				"POSTGRES_USER":     "myuser",
				"POSTGRES_PASSWORD": "s3cur3",
				"POSTGRES_DB":       "appdb",
			},
		},
	}

	broker, err := New()
	if err != nil {
		panic(err)
	}
	for _, testcase := range testcases {
		testcase := testcase
		t.Run(testcase.image, func(t *testing.T) {
			t.Parallel()
			for _, tag := range testcase.tags {
				tag := tag
				t.Run(tag, func(t *testing.T) {
					t.Parallel()
					testScopedContainer(t, broker, testcase.image, tag, testcase.tcpPorts, testcase.udpPorts, testcase.env)
				})
			}
		})
	}
}

func testScopedContainer(t *testing.T, broker scopetainer.Broker, name, tag string, tcpPorts, udpPorts []int, env map[string]string) {
	image, err := scopetainer.ImageFrom(DockerHubLibrary, name, tag)
	if err != nil {
		panic(err)
	}
	ctx, ctxCancel := context.WithTimeout(context.Background(), time.Minute*5)
	defer ctxCancel()
	container, cleanup, err := broker.Start(ctx, scopetainer.ContainerConfig{
		Image:                image,
		TCPPorts:             tcpPorts,
		EnvironmentVariables: env,
	}, scopetainer.AsTCPHealthChecks(tcpPorts, time.Minute)...)
	assert.NoError(t, err)
	for _, port := range tcpPorts {
		if assert.Contains(t, container.PortBinding, port) {
			assert.Greater(t, container.PortBinding[port], 0)
		}
	}
	if assert.NotNil(t, cleanup) {
		cleanupCtx, cleanupCtxCancel := context.WithTimeout(context.Background(), time.Minute)
		defer cleanupCtxCancel()
		err := cleanup(cleanupCtx)
		assert.NoError(t, err, "could not cleanup container: %#v", err)
	}
}
