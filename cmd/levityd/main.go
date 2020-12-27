package main

import (
	"log"
	"net"
	"os"

	"github.com/spf13/cobra"
	"github.com/tcsc/levity/api"
	"github.com/tcsc/levity/taskmanager"
	"google.golang.org/grpc"
)

var rootCmd = cobra.Command{
	Use:   "levityd [address]",
	Short: "Levity: A simple task manager service",
	Args:  cobra.ExactArgs(1),
	Run:   levityMain,
}

// The CLI for the daemon is very simple, taking the address and
// port to bind to as its single argument. This implies that server
// can only listen to a single address, which not what you'd want
// on a production server - but would require a more involved
// configuration system than is appropriate for this exercise.

func levityMain(cmd *cobra.Command, args []string) {
	addr := args[0]

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("Failed to listen on \"%s\": %v", addr, err)
	}

	// NB: Some system tests look for some output on stdout to
	// decide that the server is up and ready to receive requests.
	// This interlock method is pretty flakey, and I'd definitely be
	// looking to replace it in a live system

	log.Printf("Listening on %s", listener.Addr().String())

	taskMan := taskmanager.New()

	grpcServer := grpc.NewServer()
	api.RegisterTaskManagerServer(grpcServer, taskMan)
	log.Printf("Serving requests")
	grpcServer.Serve(listener)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
