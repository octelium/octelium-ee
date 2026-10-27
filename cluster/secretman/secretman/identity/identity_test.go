package identity

import (
	"testing"

	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/pkg/utils/ldflags"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIdentityTokenUsesCurrentTestEnvironment(t *testing.T) {
	original := ldflags.TestMode
	ldflags.TestMode = "true"
	t.Cleanup(func() { ldflags.TestMode = original })
	t.Setenv("OCTELIUM_TEST_ID_TOKEN", "first-token")
	first, err := GetIdentityToken(t.Context(), nil)
	require.NoError(t, err)
	assert.Equal(t, "first-token", first.Token)
	t.Setenv("OCTELIUM_TEST_ID_TOKEN", "rotated-token")
	second, err := GetIdentityToken(t.Context(), &enterprisev1.SecretStore{})
	require.NoError(t, err)
	assert.Equal(t, "rotated-token", second.Token)
	assert.Equal(t, "first-token", first.Token)
}
