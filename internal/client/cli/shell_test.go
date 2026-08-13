package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRequireAuth_NotLoggedIn(t *testing.T) {
	st := &state{}
	called := false

	err := requireAuth(st, func() error { called = true; return nil })
	assert.Error(t, err)
	assert.False(t, called)
}

func TestRequireAuth_LoggedIn(t *testing.T) {
	st := &state{loggedIn: true}
	called := false

	err := requireAuth(st, func() error { called = true; return nil })
	assert.NoError(t, err)
	assert.True(t, called)
}

func TestDispatch_UnknownCommand(t *testing.T) {
	err := dispatch(&state{}, "does-not-exist", nil)
	assert.Error(t, err)
}

func TestDispatch_Help(t *testing.T) {
	err := dispatch(&state{}, "help", nil)
	assert.NoError(t, err)
}

func TestDispatch_Exit(t *testing.T) {
	err := dispatch(&state{}, "exit", nil)
	assert.ErrorIs(t, err, errExit)
}

func TestDispatch_Quit(t *testing.T) {
	err := dispatch(&state{}, "quit", nil)
	assert.ErrorIs(t, err, errExit)
}

func TestDispatch_RequiresAuthForList(t *testing.T) {
	err := dispatch(&state{}, "list", nil)
	assert.Error(t, err)
}

func TestDispatch_RequiresAuthForGet(t *testing.T) {
	err := dispatch(&state{}, "get", []string{"1"})
	assert.Error(t, err)
}

func TestDispatch_RequiresAuthForAdd(t *testing.T) {
	err := dispatch(&state{}, "add", []string{"text", "meta", "data"})
	assert.Error(t, err)
}

func TestDispatch_RequiresAuthForDelete(t *testing.T) {
	err := dispatch(&state{}, "delete", []string{"1"})
	assert.Error(t, err)
}

func TestDispatch_RequiresAuthForUpdate(t *testing.T) {
	err := dispatch(&state{}, "update", []string{"1", "meta", "data"})
	assert.Error(t, err)
}

func TestDispatch_RegisterWithoutArgs(t *testing.T) {
	err := dispatch(&state{}, "register", nil)
	assert.Error(t, err)
}
