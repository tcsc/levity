package registry

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tcsc/levity/task"
)

func TestRegisterAndLookup(t *testing.T) {
	require := require.New(t)
	uut := New()

	testTask := task.New("cat", ".", nil, "/root/.ssh/id_rsa")
	id := uut.Register(testTask)
	require.NotEmptyf(string(id), "Task ID must not be empty: \"%v\"", id)

	lookupResult := uut.Lookup(id)
	require.Same(testTask, lookupResult)
}

func TestLookupNonExistantTask(t *testing.T) {
	require := require.New(t)

	uut := New()
	task := uut.Lookup("no-such-task")
	require.Nil(task)
}
