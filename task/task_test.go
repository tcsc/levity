package task

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tcsc/levity/api"
)

func TestFormatEnv(t *testing.T) {
	assert := assert.New(t)

	env := map[string]string{
		"ALPHA":   "alfa",
		"BETA":    "bravo",
		"GAMMA":   "charlie",
		"DELTA":   "delta",
		"EPSILON": "echo",
	}

	expected := []string{
		"ALPHA=alfa",
		"BETA=bravo",
		"GAMMA=charlie",
		"DELTA=delta",
		"EPSILON=echo",
	}

	actual := formatEnvironment(env)
	assert.Equal(len(actual), len(expected))
	assert.ElementsMatch(actual, expected)
}

func TestFormatEnvHandlesEmptyEnv(t *testing.T) {
	uut := formatEnvironment(map[string]string{})
	assert.Empty(t, uut)
}

func TestNonExistentBinaryIsAnError(t *testing.T) {
	assert := assert.New(t)

	uut := New("no-such-binary", "", nil)
	assert.Error(uut.Start())
}

func await(t *Task, timeout time.Duration) error {
	select {
	case <-time.After(timeout):
		return errors.New("Timed out waiting for task to complete")

	case <-t.Done():
		return nil
	}
}

func TestCaptureStdout(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	// Given a task that produces data on stdout
	uut := New(
		"echo",
		"",
		map[string]string{},
		"-n",
		"I am the very model of a modern Major-General,",
		"I've information vegetable, animal, and mineral,",
	)
	require.NoError(uut.Start())

	// when I wait for the command to complete
	require.NoError(await(uut, 1*time.Second))

	// expect that the data written to stdout is in the task's stdout buffer
	assert.Equal(
		[]byte(
			"I am the very model of a modern Major-General, "+
				"I've information vegetable, animal, and mineral,",
		),
		uut.Stdout(),
	)
	assert.Equal(0, uut.ExitCode())
}

func TestCaptureStderr(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	// Given a task that prints some text to stderr
	uut := New(
		"bash",
		"",
		map[string]string{},
		"-c",
		"1>&2 echo Errors are not in the art, but the artificers",
	)
	require.NoError(uut.Start())

	// When I let the task run to completion
	require.NoError(await(uut, 1*time.Second))

	// The stdout buffer should contain the expected data from the task
	assert.Equal(
		[]byte("Errors are not in the art, but the artificers\n"),
		uut.Stderr(),
	)
}

func TestNonZeroExitCode(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	// Given a task that we know will *NOT* respond to a SIGTERM
	// nicely, that is definitely up and running
	uut := New(
		"exit-with-two",
		"",
		map[string]string{},
	)
	require.NoError(uut.Start())
	require.NoError(await(uut, 1*time.Second))

	assert.Equal(2, uut.exitCode)
}

// sliceContains is a test helper function to check if a byte slice contains
// the content of another byte slice
func sliceContains(buffer, target []byte) bool {
	targetLength := len(target)
	for len(buffer) >= targetLength {
		found := true
		for i, b := range target {
			if buffer[i] != b {
				found = false
				break
			}
		}

		if found {
			return true
		}
		buffer = buffer[1:]
	}
	return false
}

func TestSliceContains(t *testing.T) {
	type testCase struct {
		expect bool
		buffer string
		target string
	}

	testCases := []testCase{
		{buffer: "ABCDEFG", target: "ABC", expect: true},
		{buffer: "ABCDEFG", target: "DEF", expect: true},
		{buffer: "ABCDEFG", target: "EFG", expect: true},
		{buffer: "ABCDEFG", target: "", expect: true},
		{buffer: "ABCDEFG", target: "EFGH", expect: false},
		{buffer: "", target: "ABC", expect: false},
		{buffer: "AB", target: "ABCDEF", expect: false},
	}

	for _, tc := range testCases {
		testName := fmt.Sprintf("\"%s\" in \"%s\" (%t)",
			tc.target, tc.buffer, tc.expect)

		t.Run(testName, func(t *testing.T) {
			assert.Equal(
				t,
				sliceContains([]byte(tc.buffer), []byte(tc.target)),
				tc.expect)
		})
	}
}

func TestSignal(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	// Given a task that we know will only exit when given a
	// SIGTERM that is up and running
	uut := New(
		"quit-on-sigterm",
		"",
		map[string]string{},
	)
	require.NoError(uut.Start())

	// NB: We want to make sure the task has had time to start and bind to any
	//     of the signals we want it to respond to. If we poke it too early it
	//     will be the loader responding, not the target task
	pattern := []byte("Ready")
	for !sliceContains(uut.Stdout(), pattern) {
		<-time.After(10 * time.Millisecond)
	}

	// When we signal the task to quit, with a timeout to brutally kill the
	// process after 1 sec...
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	require.NoError(uut.Signal(ctx))

	// Expect that the task will end in under the given timeout,
	// with a status indicating that it exited naturally
	require.NoError(await(uut, 2*time.Second))

	// also expect that it will have the "finished normally" status
	assert.Equal(api.TaskStatusCode_Finished, uut.statusCode)
}

func TestSignalTimeout(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	// Given a task that we know will *NOT* respond to a SIGTERM
	// nicely, that is definitely up and running
	uut := New(
		"ignore-signal",
		"",
		map[string]string{},
	)
	require.NoError(uut.Start())

	// NB: We want to make sure the task has had time to start and bind to any
	//     of the signals we want it to respond to. If we poke it too early it
	//     will be the loader responding, not the target task
	pattern := []byte("Ready")
	for !sliceContains(uut.Stdout(), pattern) {
		<-time.After(10 * time.Millisecond)
	}

	// When we signal it to quit, brutally killing the process
	// after 1 sec
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	require.NoError(uut.Signal(ctx))

	// Expect that the task will end in under the given timeout,
	// with a status indicating that it was brutally killed
	require.NoError(await(uut, 2*time.Second))

	// also expect that it will have the "brutal kill" status
	assert.Equal(api.TaskStatusCode_BrutallyKilled, uut.statusCode)
}
func TestEnvironment(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	// Given a running task with a configured environment
	uut := New(
		"sh",
		"",
		map[string]string{"FOO": "BAR"},
		"-c",
		"echo The value of FOO is $FOO",
	)
	require.NoError(uut.Start())

	// when I wait for the task to exit
	require.NoError(await(uut, 1*time.Second))

	// The task's stdout should indicate that the $FOO environment variable
	// was set.
	assert.True(sliceContains(
		uut.Stdout(), []byte("FOO is BAR")))
}

func TestWorkingDir(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	// Given a running pwd in the root directory
	uut := New(
		"pwd",
		"/",
		map[string]string{"FOO": "BAR"},
	)
	require.NoError(uut.Start())

	// when I wait for the task to exit
	require.NoError(await(uut, 1*time.Second))

	// The task's stdout should indicate that pwd was run under "/"
	assert.Equal([]byte("/\n"), uut.Stdout())
}
