package scopetainer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestImage(t *testing.T) {
	registry := "registry.hub.docker.com/library"
	name := "nginx"
	tag := "1.21"
	t.Run("all missing", func(t *testing.T) {
		_, err := ImageFrom("", "", "")
		assert.Error(t, err)
	})
	t.Run("missing name", func(t *testing.T) {
		_, err := ImageFrom(registry, "", tag)
		assert.Error(t, err)
	})
	t.Run("registry and name", func(t *testing.T) {
		image, err := ImageFrom(registry, name, tag)
		assert.NoError(t, err)
		assert.Equal(t, registry+"/"+name+":"+tag, image.String())
	})
	t.Run("name and tag", func(t *testing.T) {
		image, err := ImageFrom("", name, tag)
		assert.NoError(t, err)
		assert.Equal(t, name+":"+tag, image.String())
	})
	t.Run("missing tag", func(t *testing.T) {
		t.Run("with registry and name", func(t *testing.T) {
			image, err := ImageFrom(registry, name, "")
			assert.NoError(t, err)
			assert.Equal(t, registry+"/"+name+":latest", image.String())
		})
		t.Run("with name", func(t *testing.T) {
			image, err := ImageFrom("", name, "")
			assert.NoError(t, err)
			assert.Equal(t, name+":latest", image.String())
		})
	})
}
