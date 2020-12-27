package main

import (
	"context"
	"log"

	"github.com/spf13/cobra"
	"github.com/tcsc/levity/api"
)

var (
	cmdSignal = cobra.Command{
		Use:   "signal [task-id]",
		Short: "Signal the task to quit",
		Args:  cobra.ExactArgs(1),
		Run:   signalTask,
	}
)

func signalTask(cmd *cobra.Command, args []string) {
	request := &api.SignalTaskRequest{TaskId: &api.TaskHandle{Id: args[0]}}

	conn, client, err := makeClient()
	if err != nil {
		log.Fatalf("Failed to create GRPC client: %v", err)
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	_, err = client.SignalTask(ctx, request)
	if err != nil {
		log.Fatalf("GRPC request failed: %v", err)
	}
}
