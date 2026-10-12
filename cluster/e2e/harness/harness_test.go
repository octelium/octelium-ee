package harness

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFixtureContext(t *testing.T) {
	h := Wrap(nil)

	t.Run("ActiveTest", func(t *testing.T) {
		ctx, cancel := h.Ctx(t)
		defer cancel()
		deadline, ok := ctx.Deadline()
		require.True(t, ok)
		assert.True(t, time.Until(deadline) <= time.Minute)
		require.Nil(t, ctx.Err())

		t.Cleanup(func() {
			require.NotNil(t, t.Context().Err())
			ctx, cancel := h.Ctx(t)
			defer cancel()
			require.Nil(t, ctx.Err())
			_, ok := ctx.Deadline()
			assert.True(t, ok)
		})
	})

	t.Run("SynchronizationBudget", func(t *testing.T) {
		ctx, cancel := h.CtxWithin(t, SyncBudget)
		defer cancel()
		deadline, ok := ctx.Deadline()
		require.True(t, ok)
		assert.True(t, time.Until(deadline) > time.Minute)
		assert.True(t, time.Until(deadline) <= SyncBudget)
	})
}
