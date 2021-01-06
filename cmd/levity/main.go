package main

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io/ioutil"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/tcsc/levity/api"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
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

	serverAddress   string
	timeout         time.Duration = 5 * time.Second
	caCertPath      string
	connectInsecure bool
)

func init() {
	flags := rootCmd.PersistentFlags()

	flags.StringVarP(&serverAddress, argAddress, "a", "",
		"The server address and port")

	flags.DurationVarP(&timeout, "timeout", "t", timeout,
		"Timeout for GRPC requests")

	flags.StringVar(&caCertPath, "ca", "", "Override CA file")
	flags.BoolVar(&connectInsecure, "insecure", false,
		"Connect insecurely to the server (i.e. no TLS)")

	if err := rootCmd.MarkPersistentFlagRequired(argAddress); err != nil {
		panic(err)
	}
	rootCmd.AddCommand(&cmdStart, &cmdFetchLogs, &cmdQuery, &cmdSignal)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func makeCredentials() (grpc.DialOption, error) {
	if connectInsecure {
		return grpc.WithInsecure(), nil
	}

	config := &tls.Config{
		InsecureSkipVerify: false,
	}

	if caCertPath != "" {
		cert, err := ioutil.ReadFile(caCertPath)
		if err != nil {
			return nil, err
		}

		certPool := x509.NewCertPool()
		if !certPool.AppendCertsFromPEM(cert) {
			return nil, errors.New("Failed to append certificate")
		}

		config.RootCAs = certPool
	}

	creds := grpc.WithTransportCredentials(credentials.NewTLS(config))
	return creds, nil
}

func makeClient() (*grpc.ClientConn, api.TaskManagerClient, error) {
	creds, err := makeCredentials()
	if err != nil {
		return nil, nil, err
	}

	conn, err := grpc.Dial(serverAddress, creds)
	if err != nil {
		return nil, nil, err
	}

	return conn, api.NewTaskManagerClient(conn), nil
}
