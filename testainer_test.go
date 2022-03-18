package testainer

import (
	"context"
	"testing"

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
		return nil
	}
	err := Use(ctx, config, callback)
	assert.NoError(t, err)
	assert.True(t, callbackExecuted, "callback was not executed")
}
