package harness

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/octelium/octelium/cluster/e2e/harness"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSinkSetTokenRestoresDuringCleanup(t *testing.T) {
	dir := t.TempDir()
	require.Nil(t, os.WriteFile(filepath.Join(dir, "kubectl"), []byte(`#!/bin/sh
while [ "$1" != "--" ]; do
	shift
done
shift
exec "$@"
`), 0755))
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	sink := &Sink{
		h:         Wrap(&harness.H{}),
		Token:     "original-token",
		namespace: "e2e",
		pod:       "deployment/otel-sink",
		dir:       dir,
	}
	sink.SetToken(t, sink.Token)

	t.Run("Rotate", func(t *testing.T) {
		t.Cleanup(func() {
			require.ErrorIs(t, t.Context().Err(), context.Canceled)
			sink.SetToken(t, sink.Token)
		})
		sink.SetToken(t, "rotated-token")
		out, err := os.ReadFile(filepath.Join(dir, "auth-token"))
		require.Nil(t, err)
		assert.Equal(t, "rotated-token", string(out))
	})

	out, err := os.ReadFile(filepath.Join(dir, "auth-token"))
	require.Nil(t, err)
	assert.Equal(t, sink.Token, string(out))
}

func TestSinkStopRestoresDuringCleanup(t *testing.T) {
	for _, explicit := range []bool{false, true} {
		name := "Cleanup"
		if explicit {
			name = "Explicit"
		}
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			require.Nil(t, os.WriteFile(filepath.Join(dir, "kubectl"), []byte(`#!/bin/sh
printf '%s\n' "$*" >> "$0.calls"
`), 0755))
			t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
			sink := &Sink{h: Wrap(&harness.H{}), namespace: "e2e"}

			t.Run("Stop", func(t *testing.T) {
				restore := sink.Stop(t)
				if explicit {
					restore()
					restore()
				}
			})

			out, err := os.ReadFile(filepath.Join(dir, "kubectl.calls"))
			require.Nil(t, err)
			assert.Equal(t, []string{
				"scale -n e2e deployment/otel-sink --replicas=0",
				"scale -n e2e deployment/otel-sink --replicas=1",
				"rollout status -n e2e deployment/otel-sink --timeout=5m",
			}, strings.Split(strings.TrimSpace(string(out)), "\n"))
		})
	}
}
