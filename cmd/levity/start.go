package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tcsc/levity/api"
)

var (
	workingDir string
	envStrings []string

	cmdStart = cobra.Command{
		Use:   "start command [arg1...]",
		Short: "Start a task on the server",
		Run:   startTask,
		Args:  cobra.MinimumNArgs(1),
	}
)

func init() {
	cmdStart.Flags().StringVarP(&workingDir, "dir", "d", "",
		"Optionally set the working dir on the server")

	cmdStart.Flags().StringSliceVarP(&envStrings, "define", "D",
		[]string{},
		"Define an environment variable, in the form FOO=BAR")
}

func formatEnv(env []string) map[string]string {
	result := make(map[string]string)
	for _, s := range env {
		parts := strings.SplitN(s, "=", 2)
		name := ""
		value := ""

		if len(parts) == 0 {
			name = s
		} else {
			name = parts[0]
		}

		if len(parts) > 1 {
			value = parts[1]
		}

		if len(name) > 0 {
			result[name] = value
		}
	}
	return result
}

func startTask(cmd *cobra.Command, args []string) {
	request := &api.StartTaskRequest{
		Binary:      args[0],
		Environment: formatEnv(envStrings),
		Args:        args[1:],
	}
	if workingDir != "" {
		request.WorkingDir = &workingDir
	}

	conn, client, err := makeClient()
	if err != nil {
		log.Fatalf("Failed to create GRPC client: %v", err)
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	response, err := client.StartTask(ctx, request)
	if err != nil {
		log.Fatalf("GRPC request failed: %v", err)
	}

	fmt.Println(response.TaskId.Id)
}
