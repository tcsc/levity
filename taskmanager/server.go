package taskmanager

import (
	"context"
	"fmt"
	"time"

	"github.com/tcsc/levity/api"
	"github.com/tcsc/levity/registry"
	"github.com/tcsc/levity/task"
	"github.com/tcsc/levity/user"
	"google.golang.org/protobuf/types/known/emptypb"
)

// NoSuchTask is an error type indicating that the requested task does not
// exist
type NoSuchTask struct {
	id string
}

func (e *NoSuchTask) Error() string {
	return fmt.Sprintf("No such task: %s", e.id)
}

// AccessDenied is an error type indicating that the caller does not have
// sufficient privileges on the specified task to perform the requested
// operation
type AccessDenied struct {
	id string
}

func (e *AccessDenied) Error() string {
	return fmt.Sprintf("Access denied on task %s", e.id)
}

// Abstracts out the authorisation policy and permissions model.
type authorisationPolicy interface {
	Allows(*user.User, *task.Task) bool
}

// Implements the default authorisation policy: only the creator/owner of a
// task may interact with it
type defaultAuthPolicy struct{}

func (p defaultAuthPolicy) Allows(user *user.User, task *task.Task) bool {
	return user.Is(task.Owner())
}

// Server is an implementation of the TaskManager API.
type Server struct {
	api.UnimplementedTaskManagerServer
	registry   *registry.Registry
	authPolicy authorisationPolicy
}

// New creates and initialises a new Server with default settings
func New() *Server {
	return &Server{registry: registry.New(), authPolicy: defaultAuthPolicy{}}
}

// StartTask attempts to start and register a task with the task manager.
//
// Expects that a User instance has been injected into the context,
// representing the client's identity. Failure to include this will panic
// the goroutine.
func (server *Server) StartTask(ctx context.Context, req *api.StartTaskRequest) (*api.StartTaskResponse, error) {
	user := user.MustFromContext(ctx)

	t := task.New(
		user,
		req.GetBinary(),
		req.GetWorkingDir(),
		req.GetEnvironment(),
		req.GetArgs()...)

	// Start the task
	err := t.Start()
	if err != nil {
		return nil, err
	}

	// record it in the registry
	id := server.registry.Register(t)

	// Give the caller a handle to their task
	return &api.StartTaskResponse{
		TaskId: &api.TaskHandle{Id: id},
	}, nil
}

// FetchLogs extracts and returns the collected stdout & stderr data from the
// task
//
// Expects that a User instance has been injected into the context,
// representing the client's identity. Failure to include this will panic
// the goroutine.
func (server *Server) FetchLogs(
	ctx context.Context, req *api.FetchLogsRequest) (*api.FetchLogsResponse, error) {
	user := user.MustFromContext(ctx)
	taskID := req.TaskId.Id

	task := server.registry.Lookup(taskID)
	if task == nil {
		return nil, &NoSuchTask{id: taskID}
	}

	if !server.authPolicy.Allows(user, task) {
		return nil, &AccessDenied{id: taskID}
	}

	response := &api.FetchLogsResponse{
		Stdout: task.Stdout(),
		Stderr: task.Stderr(),
	}

	return response, nil
}

// QueryTask fetches information about a given task
//
// Expects that a User instance has been injected into the context,
// representing the client's identity. Failure to include this will panic
// the goroutine.
func (server *Server) QueryTask(
	ctx context.Context, req *api.QueryTaskRequest) (*api.QueryTaskResponse, error) {
	user := user.MustFromContext(ctx)
	taskID := req.TaskId.Id

	task := server.registry.Lookup(taskID)
	if task == nil {
		return nil, &NoSuchTask{id: taskID}
	}

	if !server.authPolicy.Allows(user, task) {
		return nil, &AccessDenied{id: taskID}
	}

	var exitCode *int32
	status, taskExitCode := task.Status()
	if status == api.TaskStatusCode_Finished {
		exitCode = new(int32)
		(*exitCode) = int32(taskExitCode)
	}

	response := &api.QueryTaskResponse{
		StatusCode: status,
		ExitCode:   exitCode,
	}

	return response, nil
}

// SignalTask requests that a task should be stopped
//
// Expects that a User instance has been injected into the context,
// representing the client's identity. Failure to include this will panic
// the goroutine.
func (server *Server) SignalTask(
	ctx context.Context, req *api.SignalTaskRequest) (*emptypb.Empty, error) {
	user := user.MustFromContext(ctx)
	taskID := req.TaskId.Id
	task := server.registry.Lookup(taskID)
	if task == nil {
		return nil, &NoSuchTask{id: taskID}
	}

	if !server.authPolicy.Allows(user, task) {
		return nil, &AccessDenied{id: taskID}
	}

	// Note the hardcoded 5s timeout here; this should at the very least be
	// a be parameter of the Server, preferably configurable somehow by the
	// user. For the sake of this exercise, it's just a hardcoded value.
	signalCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

	// Start a goroutine to monitor the task and free up the context when
	// the task finishes. We can't use the normal `defer cancel()` because the
	// task will have to live longer than this function call
	go func() {
		<-task.Done()
		cancel()
	}()

	err := task.Signal(signalCtx)
	if err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}
