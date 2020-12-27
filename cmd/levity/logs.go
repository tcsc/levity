package main

import (
	"context"
	"log"
	"os"

	"github.com/spf13/cobra"
	"github.com/tcsc/levity/api"
)

var (
	cmdFetchLogs = cobra.Command{
		Use:   "logs [task-id]",
		Short: "Fetch the task logs",
		Long:  "Fetch task output (stdout & stderr). Streams are witten to the corresponding client stream.",
		Args:  cobra.ExactArgs(1),
		Run:   fetchLogs,
	}
)

func fetchLogs(cmd *cobra.Command, args []string) {
	conn, client, err := makeClient()
	if err != nil {
		log.Fatalf("Failed to create GRPC client: %v", err)
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	request := &api.FetchLogsRequest{TaskId: &api.TaskHandle{Id: args[0]}}
	response, err := client.FetchLogs(ctx, request)
	if err != nil {
		log.Fatalf("GRPC request failed: %v", err)
	}

	os.Stdout.Write(response.Stdout)
	os.Stderr.Write(response.Stderr)
}
