package testainer

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRunContainer(t *testing.T) {
	config := Config{
		Registry: DockerHubLibraryRegistry,
		Image:    `nginx`,
		Tag:      `1.21`,
		Port:     80,
		Env: map[string]string{
			"SOMETHING": "ARBITRARY",
		},
	}
	ctx := context.Background()
	containerDetails, cleanup, err := Run(ctx, config)
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

func TestUseContainer(t *testing.T) {
	config := Config{
		Registry: DockerHubLibraryRegistry,
		Image:    `nginx`,
		Tag:      `1.21`,
		Port:     80,
		Env: map[string]string{
			"SOMETHING": "ARBITRARY",
		},
	}
	ctx := context.Background()
	callbackExecuted := false
	callback := func(ctx context.Context, containerDetails ContainerDetails) error {
		callbackExecuted = true
		assert.NotNil(t, ctx, "No context was provided for the container callback")
		assert.NotZero(t, containerDetails.Port, "We were not given a valid port to communicate over")
		assertTCPPortOpen(t, containerDetails.Port)
		return nil
	}
	err := Use(ctx, config, callback)
	assert.NoError(t, err)
	assert.True(t, callbackExecuted, "callback was not executed")
}

func assertTCPPortOpen(t *testing.T, port int) {
	t.Helper()
	connection, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), time.Second*3)
	if assert.NoError(t, err, "port was not open: %d", port) {
		assert.NotNil(t, connection, "port was not open: %d", port)
	}
}
