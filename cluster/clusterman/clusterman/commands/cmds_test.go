// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package commands

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunCommandRegistrationAndHooks(t *testing.T) {
	before := Cmd.Commands()
	t.Cleanup(func() {
		for _, command := range Cmd.Commands() {
			Cmd.RemoveCommand(command)
		}
		Cmd.AddCommand(before...)
	})
	InitCmds()
	command, args, err := Cmd.Find([]string{"run"})
	require.NoError(t, err)
	require.NotNil(t, command)
	assert.Equal(t, "run", command.Name())
	assert.Empty(t, args)
	assert.NotNil(t, command.RunE)
	require.NoError(t, Cmd.PersistentPreRunE(command, nil))
	require.NoError(t, Cmd.PersistentPostRunE(command, nil))
}
