package taskmanager

import (
	"context"

	"github.com/stretchr/testify/mock"
	"github.com/tcsc/levity/task"
)

type mockRegistry struct {
	mock.Mock
}

func (r *mockRegistry) AllocateID(ctx context.Context) (string, error) {
	rval := r.Called(ctx)
	return rval.String(0), rval.Error(1)
}

func (r *mockRegistry) BindToID(id string, t *task.Task) error {
	rval := r.Called(id, t)
	return rval.Error(0)
}

func (r *mockRegistry) Lookup(id string) *task.Task {
	rval := r.Called(id).Get(0)
	if rval != nil {
		return rval.(*task.Task)
	}
	return nil
}
