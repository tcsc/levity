package taskmanager

import (
	"context"
	"fmt"
	"time"

	"github.com/tcsc/levity/api"
	"github.com/tcsc/levity/registry"
	"github.com/tcsc/levity/task"
	"google.golang.org/protobuf/types/known/emptypb"
)

type NoSuchTask struct {
	id string
}

func (e *NoSuchTask) Error() string {
	return fmt.Sprintf("No such task: %s", e.id)
}

// Server is an implementation of the TaskManager API
type Server struct {
	api.UnimplementedTaskManagerServer
	registry *registry.Registry
}

// New creates and initialises a new Server with default settings
func New() *Server {
	return &Server{registry: registry.New()}
}

// StartTask attempts to start and register a task with the task manager.
func (server *Server) StartTask(ctx context.Context, req *api.StartTaskRequest) (*api.StartTaskResponse, error) {
	t := task.New(
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
func (server *Server) FetchLogs(
	ctx context.Context, req *api.FetchLogsRequest) (*api.FetchLogsResponse, error) {

	task := server.registry.Lookup(req.TaskId.Id)
	if task == nil {
		return nil, &NoSuchTask{id: req.TaskId.Id}
	}

	response := &api.FetchLogsResponse{
		Stdout: task.Stdout(),
		Stderr: task.Stderr(),
	}

	return response, nil
}

// QueryTask fetches information about a given task
func (server *Server) QueryTask(
	ctx context.Context, req *api.QueryTaskRequest) (*api.QueryTaskResponse, error) {

	task := server.registry.Lookup(req.TaskId.Id)
	if task == nil {
		return nil, &NoSuchTask{id: req.TaskId.Id}
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
func (server *Server) SignalTask(
	ctx context.Context, req *api.SignalTaskRequest) (*emptypb.Empty, error) {

	task := server.registry.Lookup(req.TaskId.Id)
	if task == nil {
		return nil, &NoSuchTask{id: req.TaskId.Id}
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

func killTask(t *task.Task) {
	ctx, cancel := context.WithCancel(context.Background())
	_ = t.Signal(ctx)
	cancel()
}
