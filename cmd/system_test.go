package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

var portPattern = regexp.MustCompile(`Listening on 127.0.0.1:(?P<port>\d+)`)

// portExtractor acts as an io.Writer() that scans a stream for a the
// levity daemon's "Listening on..." log message. Once the message is
// found, the port number is extracted and sent on the channel. After
// this point, the `portExtractor` reverts to being a dumb pass-through.
type portExtractor struct {
	buffer bytes.Buffer
	found  bool
	ch     chan int
	stream io.Writer
}

func (m *portExtractor) Write(p []byte) (int, error) {
	if !m.found {
		if n, err := m.buffer.Write(p); err != nil {
			return n, err
		}
		groups := portPattern.FindSubmatch(m.buffer.Bytes())
		if groups != nil {
			portText := string(groups[1])
			port, err := strconv.Atoi(string(portText))
			if err != nil {
				panic(err)
			}
			m.ch <- port
			m.found = true
			m.buffer.Reset()
		}
	}

	return m.stream.Write(p)
}

func (m *portExtractor) Close() error {
	close(m.ch)
	return nil
}

type daemon struct {
	cmd  *exec.Cmd
	port int
}

// startDaemon starts the levity daemon on the loopback address,
// wait for it to start up, and hand back a handle to it.
func startDaemon() (*daemon, error) {
	cmd := exec.Command(
		"levityd", "127.0.0.1:0",
		"--certificate", "../cert/svr-cert.pem",
		"--key", "../cert/svr-key.pem",
	)
	cmd.Stdout = os.Stdout

	stderr := &portExtractor{
		found:  false,
		ch:     make(chan int),
		stream: os.Stderr,
	}
	cmd.Stderr = stderr

	err := cmd.Start()
	if err != nil {
		return nil, err
	}

	d := daemon{cmd: cmd}

	// Wait for the server to tell us what port it is listening on.
	select {
	case port := <-stderr.ch:
		d.port = port
		return &d, nil

	case <-time.After(1 * time.Second):
		d.kill()
		return nil, errors.New("Timed out waiting for data on stdout")
	}
}

func (d *daemon) kill() {
	log.Print("Killing daemon.")
	defer log.Print("Daemon killed")
	if err := d.cmd.Process.Kill(); err != nil {
		panic(err)
	}
	_ = d.cmd.Wait()
}

func (d *daemon) addr() string {
	return fmt.Sprintf("127.0.0.1:%d", d.port)
}

// Go doesn't seem to offer a nice way to make strongly typed enumerations, but
// my pathological aversion to non-obvious boolean parameters to functions
// means that we have to go through the `connectionMode` hoopla.

type connectionMode func(args []string) []string

func secure(args []string) []string {
	return append(args, "--ca", "../cert/ca-cert.pem")
}

func insecure(args []string) []string {
	return append(args, "--insecure")
}

// runLevity executes the levity client, waiting for it to finish and returning
// the collected stdout stream.
func runLevity(connMode connectionMode, addr string, argv ...string) (string, error) {
	args := connMode([]string{"-a", addr})

	client := exec.Command("levity", append(args, argv...)...)
	output, err := client.Output()
	if err != nil {
		exitErr := err.(*exec.ExitError)
		fmt.Println(string(exitErr.Stderr))
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// levity executes the levity client command in the default (secure)
// configuration and returns the command's stdout.
func levity(addr string, argv ...string) (string, error) {
	return runLevity(secure, addr, argv...)
}

// awaitTask waits for a task on the levityd server to finish, failing
// after a given timeout. Exercises the levity `query` command to monitor
// the task running on the server.
func awaitTask(id string, daemon *daemon, timeout time.Duration) error {
	t0 := time.Now()
	for time.Since(t0) < timeout {
		stdout, err := levity(daemon.addr(), "query", id)
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

func Test_System_HappyPath(t *testing.T) {
	require := require.New(t)

	// Given a running `levityd` server
	daemon, err := startDaemon()
	require.NoError(err)
	defer daemon.kill()

	// When I start a task on the server that runs forever
	taskID, err := levity(daemon.addr(), "start", "--",
		"sh", "-c", "i=0; while true; do echo ping $i; i=`expr $i + 1`; sleep 1; done")
	require.NoError(err)

	// Expect that polling the task returns the "Running" state
	stdout, err := levity(daemon.addr(), "query", taskID)
	require.NoError(err)
	require.Equal("Running", stdout)

	<-time.After(1 * time.Second)

	// When I signal the task to quit
	_, err = levity(daemon.addr(), "signal", taskID)
	require.NoError(err)

	// Expect that the task will finish within a given time limit
	t.Log("Waiting for task to finish...")
	require.NoError(awaitTask(taskID, daemon, 5*time.Second))

	// And, finally, when I fetch the logs from the server
	t.Log("Fetching logs")
	stdout, err = levity(daemon.addr(), "logs", taskID)

	// Expect that the server has retained and returned the task's data
	require.NoError(err)
	require.True(strings.Contains(stdout, "ping 0"))
}
func Test_Client_ReturnsNonZero_OnNoSuchTask(t *testing.T) {
	require := require.New(t)

	daemon, err := startDaemon()
	require.NoError(err)
	defer daemon.kill()

	_, err = levity(daemon.addr(), "query", "no-such-task")
	require.Error(err)
	exitErr := err.(*exec.ExitError)
	require.NotEqual(0, exitErr.ExitCode())
}

func Test_Client_ReturnsNonZero_OnNoServer(t *testing.T) {
	_, err := levity("localhost:9999", "start", "ls")
	require.Error(t, err)
	exitErr := err.(*exec.ExitError)
	require.NotEqual(t, 0, exitErr.ExitCode())
}

func Test_Client_ReturnsNonZero_OnInvalidSeverCert(t *testing.T) {
	// Given a running daemon, configured with a certificate for
	// localhost & 127.0.0.1, but NOT the IPv6 loopback [::1]
	require := require.New(t)
	daemon, err := startDaemon()
	require.NoError(err)
	defer daemon.kill()

	// When I attempt to contact the server using the IPv6 lopback
	// address
	_, err = levity(fmt.Sprintf("[::]:%d", daemon.port), "start", "echo", "hello world")

	// Expect that the request fails and the client's exit code is non 0
	require.Error(err)
	exitErr := err.(*exec.ExitError)
	require.NotEqual(0, exitErr.ExitCode())

	// ... and that it was a TLS failure that caused the error
	require.Contains(
		string(exitErr.Stderr), "authentication handshake failed")
}

func Test_Client_ReturnsNonZero_OnInsecureConnection(t *testing.T) {
	// Given a levity server requiring a secured connection
	require := require.New(t)
	daemon, err := startDaemon()
	require.NoError(err)
	defer daemon.kill()

	// When I attempt to connect to it using an insecure connection
	_, err = runLevity(insecure, daemon.addr(), "start", "echo", "hello world")

	// Expect that the request fails and the client's exit code is non 0
	require.Error(err)
	exitErr := err.(*exec.ExitError)
	require.NotEqual(0, exitErr.ExitCode())
}
