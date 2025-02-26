package scenario

import (
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"testing"
	"time"
)

func TestParseEventFromYAML(t *testing.T) {
	yamlData := `
type: create
podSpec:
  name: test-pod
  namespace: default
  image: nginx:latest
  resources:
    cpu: "100m"
    memory: "128Mi"
delayAfter: 5s
`
	var evt Event
	err := yaml.Unmarshal([]byte(yamlData), &evt)
	assert.NoError(t, err, "Failed to unmarshal YAML into Event struct")

	assert.Equal(t, EventTypeCreate, evt.Type, "Event type should be create")
	assert.Equal(t, "test-pod", evt.PodSpec.Name)
	assert.Equal(t, "default", evt.PodSpec.Namespace)
	assert.Equal(t, "nginx:latest", evt.PodSpec.Image)
	assert.Equal(t, "100m", evt.PodSpec.Resources.CPU)
	assert.Equal(t, "128Mi", evt.PodSpec.Resources.Memory)
	assert.Equal(t, 5*time.Second, evt.RunAfter, "The delayAfter duration should be 5s")
}
