// Package task provides tools to manage running a process & capturing its
// output.
package task

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"os/exec"
	"sync"
	"syscall"

	"github.com/tcsc/levity/api"
)

// ErrInvalidState indicates that an operation was attempted on a task
// in a state not prepared for it
var ErrInvalidState = errors.New("task in invalid state for operation")

// streamReader catches the output from one of a Cmd's output streams (i.e.
// stdout or stderr) and writes it out to a byte buffer in a Task, under
// a write lock.
type streamReader struct {
	lock *sync.RWMutex
	dst  *bytes.Buffer
}

func (r *streamReader) Write(b []byte) (n int, err error) {
	r.lock.Lock()
	defer r.lock.Unlock()
	return r.dst.Write(b)
}

// Task represents a task that has been invoked by the API server.
type Task struct {
	lock       sync.RWMutex
	cmd        *exec.Cmd
	stdout     bytes.Buffer
	stderr     bytes.Buffer
	statusCode api.TaskStatusCode
	exitCode   int
	done       chan struct{}
}

// New creates (but does not start) new task
func New(binary string, workingDir string, env map[string]string, args ...string) *Task {
	// Construct the underlying command which will do the heavy lifting of executing
	// the subcommand.
	cmd := exec.Command(binary, args...)
	cmd.Dir = workingDir
	cmd.Env = formatEnvironment(env)

	// wrap it in a Task to provide locking, and bind the output streams to readers
	// that will capture the stream data and write it to the given buffers
	t := Task{
		cmd:        cmd,
		statusCode: api.TaskStatusCode_NotStarted,
		done:       make(chan struct{}),
		exitCode:   -1,
	}
	t.cmd.Stdout = &streamReader{lock: &t.lock, dst: &t.stdout}
	t.cmd.Stderr = &streamReader{lock: &t.lock, dst: &t.stderr}

	return &t
}

// Start starts the task running
func (t *Task) Start() error {
	t.lock.Lock()
	defer t.lock.Unlock()

	if t.statusCode != api.TaskStatusCode_NotStarted {
		return ErrInvalidState
	}

	err := t.cmd.Start()
	if err != nil {
		return err
	}

	t.statusCode = api.TaskStatusCode_Running

	// The `monitor` will wait on the underlying process to complete,
	// perform some post-exit bookeeping and then exit as well.
	go func() {
		err := t.monitor()
		if err != nil {
			log.Printf("Task failed: %v", err)

			// Move the process in to an ERROR state where all bets
			// are off.
			t.lock.Lock()
			defer t.lock.Unlock()
			t.statusCode = api.TaskStatusCode_InternalServerError
			close(t.done)
		}
	}()

	return nil
}

// Signal requests that the task quit. When the supplied context
// expires the system will give up waiting for the task to quit
// nicely and kill it.
func (t *Task) Signal(ctx context.Context) error {
	// If there was a method to promote an RLock to a write lock
	// I'd try reading the status first and only promote to a write
	// lock if it became necessary.
	t.lock.Lock()
	defer t.lock.Unlock()

	// Trying to signal a task before we start it is an error
	if t.statusCode == api.TaskStatusCode_NotStarted {
		return ErrInvalidState
	}

	// Trying to signal a task that is not in the Running state
	// is NOT an error, but we really shouldn't try signalling
	// it again; so just return
	if t.statusCode != api.TaskStatusCode_Running {
		return nil
	}

	// Signal the task to quit
	t.statusCode = api.TaskStatusCode_Signalled
	err := t.cmd.Process.Signal(syscall.SIGTERM)
	if err != nil {
		return err
	}

	go monitorSignalContext(ctx, t)

	return nil
}

func monitorSignalContext(ctx context.Context, t *Task) {
	select {
	case <-ctx.Done():
		// The context (or owner) has decided its time to stop waiting for the
		// process to die naturally, so it's now time to shoot it.
		t.brutalKill()

	case <-t.Done():
		// The task has exited naturally before the timeout expired. No
		// need to do anything other than let this goroutine exit
		// naturally
		return
	}
}

// Done returns a channel that will be closed when the task
// completed (for whatever reason), in the style of
// context.Context.
func (t *Task) Done() <-chan struct{} {
	return t.done
}

// brutalKill kills the underlying process, giving no chance for cleanup. This
// should only ever be called in response to a call to `Signal()`, and only
// then when the context is cancelled.
func (t *Task) brutalKill() {
	t.lock.Lock()
	defer t.lock.Unlock()

	// It's possible that the process has actually finished naturally by
	// the time we get here, so if the process is not still in the
	// signalled state then we don't touch it.
	if t.statusCode != api.TaskStatusCode_Signalled {
		return
	}

	t.statusCode = api.TaskStatusCode_BrutallyKilled
	err := t.cmd.Process.Kill()
	if err != nil {
		// Seems a bit excessive to panic here; The process just may have
		// exited between us deciding to kill it and us actually doing it.
		// For now we will just log the error so we have some visibility
		// and move on.
		log.Printf("Failed to kill task: %s", err.Error())
	}
}

func cloneSlice(src []byte) []byte {
	result := make([]byte, len(src))
	copy(result, src)
	return result
}

// Stdout creates and returns a copy of the current stdout data.
func (t *Task) Stdout() []byte {
	t.lock.RLock()
	defer t.lock.RUnlock()
	return cloneSlice(t.stdout.Bytes())
}

// Stderr creates and returns a copy of the current stderr data.
func (t *Task) Stderr() []byte {
	t.lock.RLock()
	defer t.lock.RUnlock()
	return cloneSlice(t.stderr.Bytes())
}

// monitor is executed in a goroutine and moves the process exit code (in the
// form of an *os.Process) into place when the underlying process has exited
func (t *Task) monitor() error {
	// Wait for the underying process to complete before we try and force all
	// of the IO streams to be flushed and closed, otherwise we have a data
	// race on the Cmd that we wrap.

	var exitCode int
	switch err := t.cmd.Wait().(type) {
	case nil:
		exitCode = 0

	case *exec.ExitError:
		// For our purposes, this counts as a clean exit
		exitCode = err.ProcessState.ExitCode()

	default:
		return err
	}

	// Now that we *know* the underlying process has finished, we can clean
	// up the Cmd while we have it locked, averting the data race
	t.lock.Lock()
	defer t.lock.Unlock()

	if t.statusCode != api.TaskStatusCode_BrutallyKilled {
		t.statusCode = api.TaskStatusCode_Finished
	}
	t.exitCode = exitCode
	close(t.done)

	return nil
}

func formatEnvironment(env map[string]string) []string {
	result := make([]string, 0, len(env))
	for k, v := range env {
		result = append(result, fmt.Sprintf("%s=%s", k, v))
	}
	return result
}

// ExitCode returns the exit code for the task process, or -1 if it is either
// still running or was brutally killed.
func (t *Task) ExitCode() int {
	t.lock.RLock()
	defer t.lock.RUnlock()
	return t.exitCode
}
