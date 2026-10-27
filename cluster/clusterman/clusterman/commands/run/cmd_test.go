package run

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRunRejectsUnavailableKubernetesConfiguration(t *testing.T) {
	t.Setenv("KUBECONFIG", filepath.Join(t.TempDir(), "missing-config"))
	t.Setenv("KUBERNETES_SERVICE_HOST", "")
	t.Setenv("KUBERNETES_SERVICE_PORT", "")
	assert.Error(t, doCmd(Cmd, nil))
}
