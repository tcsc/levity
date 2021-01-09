package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io/ioutil"
	"log"
	"net"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/tcsc/levity/api"
	"github.com/tcsc/levity/taskmanager"
	"github.com/tcsc/levity/user"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
)

var (
	rootCmd = cobra.Command{
		Use:   "levityd [address]",
		Short: "Levity: A simple task manager service",
		Args:  cobra.ExactArgs(1),
		Run:   levityMain,
	}

	clientCACertPath string
	certificatePath  string
	privateKeyPath   string
)

func init() {
	rootCmd.Flags().StringVarP(&certificatePath, "certificate", "c",
		"./cert/svr-cert.pem",
		"Path to server TLS certificate")

	rootCmd.Flags().StringVarP(&privateKeyPath, "key", "k",
		"./cert/svr-key.pem",
		"Path to server private key")

	rootCmd.Flags().StringVar(&clientCACertPath, "client-ca",
		"",
		"Specify the root CA used to validate client certificates")
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

func initTLS() (grpc.ServerOption, error) {

	log.Printf("Loading TLS certificate from %s", certificatePath)
	log.Printf("Loading private key from %s", privateKeyPath)
	log.Printf("Client CA Cert path: %s", clientCACertPath)

	idCert, err := tls.LoadX509KeyPair(certificatePath, privateKeyPath)
	if err != nil {
		return nil, err
	}

	tlsCfg := &tls.Config{
		Certificates: []tls.Certificate{idCert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		MinVersion:   tls.VersionTLS13,
	}

	if clientCACertPath != "" {
		cert, err := ioutil.ReadFile(clientCACertPath)
		if err != nil {
			return nil, err
		}

		certPool := x509.NewCertPool()
		if !certPool.AppendCertsFromPEM(cert) {
			return nil, errors.New("Failed to append certificate")
		}

		tlsCfg.ClientCAs = certPool
	}

	return grpc.Creds(credentials.NewTLS(tlsCfg)), nil
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
	creds, err := initTLS()
	if err != nil {
		log.Fatalf("Failed to configure TLS : %v", err)
	}
	options = append(options, creds, grpc.UnaryInterceptor(authenticateRequest))

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

// Maps the certificate supplied by the user to a levity daemon user account
// and injects that user into the context supplied to the handler.
func authenticateRequest(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler) (interface{}, error) {

	p, ok := peer.FromContext(ctx)
	if !ok {
		// TODO: make a more useful/descriptive error type
		return nil, errors.New("No Peer")
	}

	tlsInfo, ok := p.AuthInfo.(credentials.TLSInfo)
	if !ok {
		// TODO: make a more useful/descriptive error type
		return nil, errors.New("No TLS Info")
	}

	if len(tlsInfo.State.PeerCertificates) == 0 {
		// Not sure if it's even possible to get here, as the TLS configuration
		// requires a client certificate to be verified before this is even
		// called. Probably good to check here anyway, though, in case the TLS
		// config changes.
		return nil, errors.New("No client certificate")
	}
	clientCertificate := tlsInfo.State.PeerCertificates[0]

	// In a real application, a lot more validation would need to go here (e.g.
	// checking a user database, consulting certificate revocation lists,
	// figuring out the claims the user has on the server, etc). The way a
	// certificate is mapped to a user would *definitely* need to be more
	// sophisticated.
	//
	// You might even go so far as to introduce a second `authorisation`
	// service to do all this once and issue a token of some sort that that
	// can be reused without having to reload it every call.
	//
	// But for this exercise, for the sake of simplicity, we're just going to
	// assume that
	//  1. Holding a valid certificate implies you are an authorised user, and
	//  2. The certificate CN is a unique identifier for user on the system.
	//
	// (NB: I would *not* consider these good assumptionions for a production
	//	system)
	loginName := clientCertificate.Subject.CommonName
	ctxWithUser := user.NewContext(ctx, user.New(loginName))
	return handler(ctxWithUser, req)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
