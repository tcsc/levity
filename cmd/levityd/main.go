package main

import (
	"log"
	"net"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/tcsc/levity/api"
	"github.com/tcsc/levity/taskmanager"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var (
	rootCmd = cobra.Command{
		Use:   "levityd [address]",
		Short: "Levity: A simple task manager service",
		Args:  cobra.ExactArgs(1),
		Run:   levityMain,
	}

	serveInsecure   bool
	certificatePath string
	privateKeyPath  string
)

func init() {
	rootCmd.Flags().StringVarP(&certificatePath, "certificate", "c",
		"./cert/svr-cert.pem",
		"Path to TLS certificate")

	rootCmd.Flags().StringVarP(&privateKeyPath, "key", "k",
		"./cert/svr-key.pem",
		"Path to server private key")

	rootCmd.Flags().BoolVar(&serveInsecure, "insecure", false,
		"Serve over an unsecured connection")
}

func expandPaths() error {
	s, err := filepath.Abs(certificatePath)
	if err != nil {
		return err
	}
	certificatePath = s

	s, err = filepath.Abs(privateKeyPath)
	if err != nil {
		return err
	}
	privateKeyPath = s

	return nil
}

// The CLI for the daemon is very simple, taking the address and
// port to bind to as its single argument. This implies that server
// can only listen to a single address, which not what you'd want
// on a production server - but would require a more involved
// configuration system than is appropriate for this exercise.

func levityMain(cmd *cobra.Command, args []string) {
	addr := args[0]

	if err := expandPaths(); err != nil {
		log.Fatalf("Failed to get absolute paths for TLS keys: %v", err)
	}

	options := make([]grpc.ServerOption, 0, 1)
	if !serveInsecure {
		log.Printf("Loading TLS certificate from %s", certificatePath)
		log.Printf("Loading private key from %s", privateKeyPath)
		creds, err := credentials.NewServerTLSFromFile(certificatePath, privateKeyPath)
		if err != nil {
			log.Fatalf("Failed to load server TLD certificate: %v", err)
		}
		options = append(options, grpc.Creds(creds))
	}

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("Failed to listen on \"%s\": %v", addr, err)
	}

	// NB: Some system tests look for the following line to know where
	//     to point their clients. This interlock method is pretty
	//     flakey, and I'd definitely be looking to replace it in a
	//     live system
	log.Printf("Listening on %s", listener.Addr().String())

	taskMan := taskmanager.New()

	grpcServer := grpc.NewServer(options...)
	api.RegisterTaskManagerServer(grpcServer, taskMan)
	log.Printf("Serving requests")
	err = grpcServer.Serve(listener)
	if err != nil {
		log.Fatalf("GRPC server faled to start: %v", err)
	}
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
