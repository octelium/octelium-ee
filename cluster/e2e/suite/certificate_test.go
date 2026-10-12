package suite

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPeerCertificateContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := peerCertificate(ctx, "127.0.0.1")
	require.ErrorIs(t, err, context.Canceled)
}
