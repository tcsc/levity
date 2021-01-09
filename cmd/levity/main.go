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
	argAddress        = "address"
	argClientCert     = "certificate"
	argClientKey      = "key"
	argUseObsoleteTLS = "use-obsolete-tls"
)

var (
	rootCmd = cobra.Command{
		Use:   "levity",
		Short: "Levity: A simple task manager service",
		Long:  "Levity is a remote task runner",
	}

	serverAddress  string
	timeout        time.Duration = 5 * time.Second
	caCertPath     string
	idPrivateKey   string
	idUserCert     string
	useObsoleteTLS bool
)

func init() {
	flags := rootCmd.PersistentFlags()

	flags.StringVarP(&serverAddress, argAddress, "a", "",
		"The server address and port")

	flags.DurationVarP(&timeout, "timeout", "t", timeout,
		"Timeout for GRPC requests")

	flags.StringVar(&caCertPath, "ca", "", "Override system CA with provided CA")

	flags.StringVarP(&idPrivateKey, argClientKey, "k", "",
		"Path to TLS identity private key")

	flags.StringVarP(&idUserCert, argClientCert, "c", "",
		"Path to TLS identity certificate")

	for _, arg := range []string{argAddress, argClientCert, argClientKey} {
		if err := rootCmd.MarkPersistentFlagRequired(arg); err != nil {
			panic(err)
		}
	}

	// Useful for testing, but should not be advertised to the user
	flags.BoolVar(&useObsoleteTLS, argUseObsoleteTLS, false,
		"Force use of TLS < 1.3 for testing")
	if err := flags.MarkHidden(argUseObsoleteTLS); err != nil {
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
	idCert, err := tls.LoadX509KeyPair(idUserCert, idPrivateKey)
	if err != nil {
		return nil, err
	}

	tlsCfg := &tls.Config{
		Certificates: []tls.Certificate{idCert},
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

		tlsCfg.RootCAs = certPool
	}

	if useObsoleteTLS {
		tlsCfg.MaxVersion = tls.VersionTLS12
	}

	creds := grpc.WithTransportCredentials(credentials.NewTLS(tlsCfg))
	return creds, nil
}

func makeClient() (*grpc.ClientConn, api.TaskManagerClient, error) {
	creds, err := makeCredentials()
	if err != nil {
		return nil, nil, err
	}

	conn, err := grpc.Dial(
		serverAddress,
		creds)
	if err != nil {
		return nil, nil, err
	}

	return conn, api.NewTaskManagerClient(conn), nil
}
