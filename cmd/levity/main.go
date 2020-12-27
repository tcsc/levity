package main

import (
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/tcsc/levity/api"
	"google.golang.org/grpc"
)

const (
	argAddress = "address"
)

var (
	rootCmd = cobra.Command{
		Use:   "levity",
		Short: "Levity: A simple task manager service",
		Long:  "Levity is a remote task runner",
	}

	serverAddress string

	timeout time.Duration = 5 * time.Second
)

func init() {
	rootCmd.PersistentFlags().StringVarP(&serverAddress, argAddress, "a", "",
		"The server address and port")

	rootCmd.PersistentFlags().DurationVarP(&timeout, "timeout", "t", timeout,
		"Timeout for GRPC requests")

	rootCmd.MarkFlagRequired(argAddress)

	rootCmd.AddCommand(&cmdStart, &cmdFetchLogs, &cmdQuery, &cmdSignal)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func makeClient() (*grpc.ClientConn, api.TaskManagerClient, error) {
	conn, err := grpc.Dial(serverAddress, grpc.WithInsecure())
	if err != nil {
		return nil, nil, err
	}

	return conn, api.NewTaskManagerClient(conn), nil
}
