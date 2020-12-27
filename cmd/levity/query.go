package main

import (
	"context"
	"fmt"
	"log"

	"github.com/spf13/cobra"
	"github.com/tcsc/levity/api"
)

var (
	cmdQuery = cobra.Command{
		Use:   "query [task-id]",
		Short: "Fetch the task status",
		Args:  cobra.ExactArgs(1),
		Run:   queryStatus,
	}
)

func queryStatus(cmd *cobra.Command, args []string) {
	request := &api.QueryTaskRequest{
		TaskId: &api.TaskHandle{Id: args[0]},
	}

	conn, client, err := makeClient()
	if err != nil {
		log.Fatalf("Failed to create GRPC client: %v", err)
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	response, err := client.QueryTask(ctx, request)
	if err != nil {
		log.Fatalf("GRPC request failed: %v", err)
	}

	fmt.Println(response.StatusCode)
	if response.StatusCode == api.TaskStatusCode_Finished {
		fmt.Println(*response.ExitCode)
	}
}
