package basic

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/paul-nelson-baker/go-testainer"
	"github.com/stretchr/testify/assert"
)

var (
	nginxConfig = testainer.Config{
		Registry: testainer.DockerHubLibraryRegistry,
		Image:    `nginx`,
		Tag:      `1.21`,
		Port:     80,
	}
)

func TestRun(t *testing.T) {
	ctx := context.Background()
	containerDetails, cleanup, err := Run(ctx, nginxConfig)
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	if !assert.NotNil(t, cleanup, "cleanup callback fn was nil") {
		t.FailNow()
	}
	defer cleanup()
	assert.NotZero(t, containerDetails.Port, "We were not given a valid port to communicate over")
	assertTCPPortOpen(t, containerDetails.Port)
}

func TestUse(t *testing.T) {
	ctx := context.Background()
	callbackExecuted := false
	var callback = func(ctx context.Context, containerDetails *testainer.ContainerDetails) error {
		callbackExecuted = true
		assert.NotNil(t, ctx, "No context was provided for the container callback")
		assert.NotZero(t, containerDetails.Port, "We were not given a valid port to communicate over")
		assertTCPPortOpen(t, containerDetails.Port)
		return nil
	}
	err := Use(ctx, nginxConfig, callback)
	assert.NoError(t, err)
	assert.True(t, callbackExecuted, "callback was not executed")
}

func TestRun_ZeroPort(t *testing.T) {
	ctx := context.Background()
	config := nginxConfig
	config.Port = 0
	_, cleanup, err := Run(ctx, config)
	assert.ErrorContains(t, err, "port must be a non-negative integer")
	if !assert.Nil(t, cleanup) {
		defer cleanup()
	}
}

func assertTCPPortOpen(t *testing.T, port int) {
	t.Helper()
	connection, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), time.Second*3)
	if assert.NoError(t, err, "port was not open: %d", port) {
		assert.NotNil(t, connection, "port was not open: %d", port)
	}
}
