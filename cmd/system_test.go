package cmd

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type daemon struct {
	cmd  *exec.Cmd
	addr string
}

type outputMonitor struct {
	first  bool
	ch     chan interface{}
	stream io.Writer
}

func (m *outputMonitor) Write(p []byte) (int, error) {
	if m.first {
		close(m.ch)
		m.first = false
	}

	return m.stream.Write(p)
}

func startDaemon(port int) (*daemon, error) {
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	cmd := exec.Command("levityd", addr)
	cmd.Stdout = os.Stdout

	stderr := &outputMonitor{
		first:  true,
		ch:     make(chan interface{}),
		stream: os.Stderr,
	}
	cmd.Stderr = stderr

	err := cmd.Start()
	if err != nil {
		return nil, err
	}

	d := daemon{
		cmd:  cmd,
		addr: addr,
	}

	// Wait for some output from the daemon before returning, in order to make
	// sure that it is up and going.
	select {
	case <-stderr.ch:
		return &d, nil

	case <-time.After(1 * time.Second):
		d.kill()
		return nil, errors.New("Timed out waiting for data on stdout")
	}
}

func (d *daemon) kill() {
	log.Print("Killing daemon.")
	defer log.Print("Daemon killed")
	d.cmd.Process.Kill()
	d.cmd.Wait()
}

// levity executes the levity client command and returns
// the command's stdout.
func levity(argv ...string) (string, error) {
	client := exec.Command("levity", argv...)
	output, err := client.Output()
	if err != nil {
		exitErr := err.(*exec.ExitError)
		fmt.Println(string(exitErr.Stderr))
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func awaitTask(id string, daemon *daemon, timeout time.Duration) error {
	t0 := time.Now()
	for time.Since(t0) < timeout {
		stdout, err := levity("-a", daemon.addr, "query", id)
		if err != nil {
			return err
		}

		lines := strings.Split(stdout, "\n")
		if strings.TrimSpace(lines[0]) == "Finished" {
			return nil
		}
		<-time.After(1 * time.Second)
	}
	return errors.New("Timed out waiting for task to complete")
}

func TestSystem(t *testing.T) {
	require := require.New(t)

	daemon, err := startDaemon(4321)
	require.NoError(err)
	defer daemon.kill()

	taskID, err := levity("-a", daemon.addr, "start", "--",
		"sh", "-c", "i=0; while true; do echo ping $i; i=`expr $i + 1`; sleep 1; done")
	require.NoError(err)

	stdout, err := levity("-a", daemon.addr, "query", taskID)
	require.NoError(err)
	require.Equal("Running", stdout)

	<-time.After(1 * time.Second)

	_, err = levity("-a", daemon.addr, "signal", taskID)
	require.NoError(err)
	t.Log("Waiting for task to finish...")
	require.NoError(awaitTask(taskID, daemon, 5*time.Second))

	t.Log("Fetching logs")
	stdout, err = levity("-a", daemon.addr, "logs", taskID)
	require.NoError(err)
	require.True(strings.Contains(stdout, "ping 0"))
}
func Test_Client_ReturnsNonZeroNoSuchTask(t *testing.T) {
	require := require.New(t)

	daemon, err := startDaemon(4321)
	require.NoError(err)
	defer daemon.kill()

	_, err = levity("-a", daemon.addr, "query", "no-such-task")
	require.Error(err)
	exitErr := err.(*exec.ExitError)
	require.NotEqual(0, exitErr.ExitCode())
}

func Test_Client_ReturnsNonZeroOnNoServer(t *testing.T) {
	_, err := levity("-a", "localhost:9999", "start", "ls")
	require.Error(t, err)
	exitErr := err.(*exec.ExitError)
	require.NotEqual(t, 0, exitErr.ExitCode())
}
