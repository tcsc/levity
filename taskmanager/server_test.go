package taskmanager

// Test cases for the Task Manager API implementation.
//
// These test cases reach inside the public API of the Task Manager and
// examine its innards more than I'd like. In nearly all cases, though,
// using a mocked-out internals was more verbose and less obvious than
// just examining the state of the Server instance.
//
// For a more serious project I'd be spending more time on making the
// mocking easier to use.

import (
	"context"
	"errors"
	"os"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/tcsc/levity/api"
	"github.com/tcsc/levity/task"
	"github.com/tcsc/levity/user"
)

var (
	alice = user.New("alice")
	bob   = user.New("bob")
)

func await(t *task.Task, timeout time.Duration) error {
	select {
	case <-time.After(timeout):
		return errors.New("Timed out waiting for task to complete")

	case <-t.Done():
		return nil
	}
}

func startTask(binary string, args ...string) *api.StartTaskRequest {
	return &api.StartTaskRequest{
		Binary:      binary,
		WorkingDir:  nil,
		Environment: make(map[string]string),
		Args:        args,
	}
}

func Test_StartTask_Success(t *testing.T) {
	require := require.New(t)
	tempDir := t.TempDir()
	target := path.Join(tempDir, "target")

	// Given a TaskManager instance
	uut := New()

	// When I issue a request to start a task...
	request := startTask("touch", target)
	userCtx := user.NewContext(context.Background(), alice)
	response, err := uut.StartTask(userCtx, request)

	// Expect that the request succeeds and that the reponse contains an
	// ID that can be used to reference the task
	require.NoError(err)
	taskID := response.TaskId.Id
	require.NotEmpty(taskID)

	// ... and that the task started an underlying process on the system,
	// which we deduce from the side-effect of the file existing)
	task := uut.registry.Lookup(taskID)
	require.NoError(await(task, 1*time.Second))
	_, err = os.Stat(target)
	require.NoErrorf(err, "target file %s must exist", target)
}

func Test_StartTask_CommandFailure(t *testing.T) {
	require := require.New(t)

	// Given a TaskManager instance
	registry := new(mockRegistry)
	registry.
		On("AllocateID", mock.Anything).Return("banana", nil)
	uut := New()

	// When I issue a request to start a task targeting a binary
	// that doesn't exist...
	ctx := user.NewContext(context.Background(), alice)
	request := startTask("/no-such-binary")
	response, err := uut.StartTask(ctx, request)

	// Expect that the request fails
	require.Error(err)
	require.Nil(response)

	// ..and that nothing was added to the task registry
	require.Equal(0, uut.registry.Len())
}

func Test_QueryTask_Signalled(t *testing.T) {
	require := require.New(t)
	ctx := user.NewContext(context.Background(), alice)

	// Given a task manager with a finished task
	uut := New()
	startResponse, err := uut.StartTask(
		ctx,
		startTask("exit-with-two"))
	require.NoError(err)
	taskID := startResponse.TaskId
	task := uut.registry.Lookup(taskID.Id)
	defer killTask(task)
	require.NoError(await(task, 1*time.Second))

	// When I query the status of that task
	status, err := uut.QueryTask(ctx, &api.QueryTaskRequest{TaskId: taskID})

	// ... expect that the status has moved to "finished" and the exit code set
	require.NoError(err)
	require.Equal(status.StatusCode, api.TaskStatusCode_Finished)
	require.NotNil(status.ExitCode)
	require.Equal(*status.ExitCode, int32(2))
}

func Test_QueryTask_NonSignalled(t *testing.T) {
	require := require.New(t)
	ctx := user.NewContext(context.Background(), alice)

	// Given a task manager with a running task
	uut := New()
	startResponse, err := uut.StartTask(
		ctx,
		startTask("exit-with-two"))
	require.NoError(err)
	taskID := startResponse.TaskId
	runningTask := uut.registry.Lookup(taskID.Id)
	defer killTask(runningTask)

	// When I query the task status
	status, err := uut.QueryTask(ctx, &api.QueryTaskRequest{TaskId: taskID})
	require.NoError(err)

	// ... expect that the task is in the "running" state and the exit code is
	// null
	require.Equal(status.StatusCode, api.TaskStatusCode_Running)
	require.Nil(status.ExitCode)

	// When I signal the task
	_, err = uut.SignalTask(ctx, &api.SignalTaskRequest{TaskId: taskID})
	require.NoError(err)

	// ... expect that the task should have moved to the "signalled" state
	status, err = uut.QueryTask(ctx, &api.QueryTaskRequest{TaskId: taskID})
	require.NoError(err)
	require.Equal(status.StatusCode, api.TaskStatusCode_Signalled)
	require.Nil(status.ExitCode)

	// And, finally, when I wait for the task to exit
	require.NoError(await(runningTask, 2*time.Second))

	// ... expect that the status has moved to "finished", with an exit
	// code of -1.
	status, err = uut.QueryTask(ctx, &api.QueryTaskRequest{TaskId: taskID})
	require.NoError(err)
	require.Equal(status.StatusCode, api.TaskStatusCode_Finished)
	require.Equal(*status.ExitCode, task.InvalidExitCode)
}

func Test_QueryTask_SomeoneElsesTask(t *testing.T) {
	require := require.New(t)
	ctxAlice := user.NewContext(context.Background(), alice)
	ctxBob := user.NewContext(context.Background(), bob)

	// Given a task manager with a running task started by Alice
	uut := New()
	startResponse, err := uut.StartTask(
		ctxAlice,
		startTask("exit-with-two"))
	require.NoError(err)
	taskID := startResponse.TaskId
	runningTask := uut.registry.Lookup(taskID.Id)
	defer killTask(runningTask)

	// When Bob queries the task status
	status, err := uut.QueryTask(ctxBob, &api.QueryTaskRequest{TaskId: taskID})

	// expect the request to fail with a "access denied" error
	require.IsType(&AccessDenied{}, err)

	// ...and that we didn't leak anything
	require.Nil(status)
}

func Test_FetchOutput(t *testing.T) {
	ctx := user.NewContext(context.Background(), alice)
	require := require.New(t)

	// Given a server with a task that has generated data on
	// stdout and stderr
	uut := New()
	startResponse, err := uut.StartTask(
		ctx,
		startTask(
			"sh",
			"-c",
			"while true; do echo this is stdout; 1>&2 echo this is stderr; done"))
	require.NoError(err)
	defer killTask(uut.registry.Lookup(startResponse.TaskId.Id))

	// try for 5s to get enough data to test
	t0 := time.Now()
	for time.Since(t0) < (5 * time.Second) {
		logResponse, err := uut.FetchLogs(
			ctx,
			&api.FetchLogsRequest{TaskId: startResponse.TaskId},
		)
		require.NoError(err)

		stdout := string(logResponse.Stdout)
		stderr := string(logResponse.Stderr)
		if strings.Contains(stdout, "this is stdout") &&
			strings.Contains(stderr, "this is stderr") {
			// Pass!
			return
		}

		<-time.After(10 * time.Millisecond)
	}

	// If we get to here, then the test is a wite-off
	require.FailNow("Expected stream content not found")
}

func Test_FetchOutput_SomeoneElsesTask(t *testing.T) {
	require := require.New(t)

	ctxAlice := user.NewContext(context.Background(), alice)
	ctxBob := user.NewContext(context.Background(), bob)

	// Given a server with a task started by Alice that has
	// generated data on stdout and stderr
	uut := New()
	startResponse, err := uut.StartTask(
		ctxAlice,
		startTask(
			"sh",
			"-c",
			"while true; do echo this is stdout; 1>&2 echo this is stderr; done"))
	require.NoError(err)
	defer killTask(uut.registry.Lookup(startResponse.TaskId.Id))

	// When Bob attempts to fetch the output....
	logResponse, err := uut.FetchLogs(
		ctxBob,
		&api.FetchLogsRequest{TaskId: startResponse.TaskId},
	)

	// expect the request to fail with a "access denied" error
	require.IsType(&AccessDenied{}, err)

	// ...and that we didn't leak anything
	require.Nil(logResponse)
}

func Test_FetchOutput_NonExistantTask(t *testing.T) {
	ctx := user.NewContext(context.Background(), alice)
	require := require.New(t)

	// Given a server with no running tasks
	uut := New()

	// When I poll a given task's logs...
	logResponse, err := uut.FetchLogs(
		ctx,
		&api.FetchLogsRequest{TaskId: &api.TaskHandle{Id: "none-such"}},
	)

	// expect the operation to fail, and no logs returned
	require.Error(err)
	require.Nil(logResponse)
}

func Test_Signal(t *testing.T) {
	require := require.New(t)
	ctx := user.NewContext(context.Background(), alice)

	// Given a task that will run forever...
	uut := New()
	startResponse, err := uut.StartTask(
		ctx,
		startTask(
			"sh",
			"-c",
			"while true; do date; sleep 5; done"))
	require.NoError(err)
	taskID := startResponse.TaskId
	task := uut.registry.Lookup(taskID.Id)

	// When I signal the task to quit
	_, err = uut.SignalTask(ctx, &api.SignalTaskRequest{TaskId: taskID})

	// expect the request to succeed
	require.NoError(err)

	// .. and expect the underlying task to exit
	require.NoError(await(task, 1*time.Second))
}

func Test_Signal_SomeoneElsesTask(t *testing.T) {
	require := require.New(t)
	ctxAlice := user.NewContext(context.Background(), alice)
	ctxBob := user.NewContext(context.Background(), bob)

	// Given a task started by Alice that will run forever...
	uut := New()
	startResponse, err := uut.StartTask(
		ctxAlice,
		startTask(
			"sh",
			"-c",
			"while true; do date; sleep 5; done"))
	require.NoError(err)
	taskID := startResponse.TaskId
	task := uut.registry.Lookup(taskID.Id)
	defer killTask(task)

	// When Bob attempts to signal the task to quit
	_, err = uut.SignalTask(ctxBob, &api.SignalTaskRequest{TaskId: taskID})

	// expect the request to fail with a "access denied" error
	require.IsType(&AccessDenied{}, err)
}
func Test_Signal_NonExistantTask(t *testing.T) {
	require := require.New(t)
	ctx := user.NewContext(context.Background(), alice)

	// Given an empty task manager
	uut := New()

	// When I signal the task to quit
	_, err := uut.SignalTask(ctx, &api.SignalTaskRequest{
		TaskId: &api.TaskHandle{Id: "none-such"}})

	// expect the request to fail with the no such task error
	require.IsType(&NoSuchTask{}, err)
}

func Test_Signal_AlreadyFinishedTask(t *testing.T) {
	require := require.New(t)
	ctx := user.NewContext(context.Background(), alice)

	// Given a task that has already run and finished
	uut := New()
	startResponse, err := uut.StartTask(
		ctx,
		startTask(
			"sh",
			"-c",
			"echo Hello world"))
	require.NoError(err)
	taskID := startResponse.TaskId
	task := uut.registry.Lookup(taskID.Id)
	require.NoError(await(task, 1*time.Second))

	// When I signal the task to quit
	_, err = uut.SignalTask(ctx, &api.SignalTaskRequest{TaskId: taskID})

	// expect the request to succeed
	require.NoError(err)
}

func Test_Signal_AlreadySignalledTask(t *testing.T) {
	require := require.New(t)
	ctx := user.NewContext(context.Background(), alice)
	uut := New()

	// Given a server that is managing a task which traps signals...
	startResponse, err := uut.StartTask(
		ctx,
		startTask(
			"sh",
			"-c",
			"trap \"echo Haha! Nope\" TERM; while true; do date; sleep 1; done"))
	require.NoError(err)
	taskID := startResponse.TaskId
	task := uut.registry.Lookup(taskID.Id)

	// (wait until we get some data on stdout to be sure the task is up and
	//  running)
	for len(task.Stdout()) == 0 {
		<-time.After(10 * time.Millisecond)
	}

	// When I signal the task and wait for it to show up in the stdout
	_, err = uut.SignalTask(ctx, &api.SignalTaskRequest{TaskId: taskID})
	require.NoError(err)
	stdout := ""
	t0 := time.Now()
	for !strings.Contains(stdout, "Haha! Nope") {
		if time.Since(t0) > (5 * time.Second) {
			require.FailNow("Timed out waiting for data")
		}
		<-time.After(500 * time.Millisecond)
		stdout = string(task.Stdout())
	}

	// ... and then signal it again
	_, err = uut.SignalTask(ctx, &api.SignalTaskRequest{TaskId: taskID})

	// expect the request to succeed
	require.NoError(err)
}

func killTask(t *task.Task) {
	ctx, cancel := context.WithCancel(context.Background())
	_ = t.Signal(ctx)
	cancel()
}
