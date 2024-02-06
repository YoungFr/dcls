package main

import (
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/youngfr/dcls/internal/auth"
	dclslog "github.com/youngfr/dcls/internal/log"
	"github.com/youngfr/dcls/internal/logserver"
	"github.com/youngfr/dcls/internal/tlscfg"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const logStoringDir = "log-services"

func main() {
	lis, err := net.Listen("tcp", "127.0.0.1:8080")
	if err != nil {
		log.Fatalf("failed to listen: %v\n", err)
	}

	_, err = os.Stat(logStoringDir)
	if os.IsNotExist(err) {
		if err = os.Mkdir(logStoringDir, 0744); err != nil {
			log.Fatalf("failed to create log storing directory: %v\n", err)
		}
	}

	clog, err := dclslog.NewLog(logStoringDir, dclslog.Config{})
	if err != nil {
		log.Fatalf("failed to create Log object: %v\n", err)
	}

	serverTLSConfig, err := tlscfg.SetupTLSConfig(tlscfg.TLSConfig{
		IsServerConfig:  true,
		EnableMutualTLS: true,
		CertFile:        tlscfg.ServerCertFile,
		KeyFile:         tlscfg.ServerKeyFile,
		CAFile:          tlscfg.CAFile,
		ServerName:      lis.Addr().String(),
	})
	if err != nil {
		log.Fatalf("failed to setup server mTLS: %v\n", err)
	}
	serverCredentials := credentials.NewTLS(serverTLSConfig)

	server, err := logserver.NewgRPCServer(
		&logserver.LogImplConfig{
			CommitLog:  clog,
			Authorizer: auth.New(tlscfg.ACLModelFile, tlscfg.ACLPolicyFile),
		},
		grpc.Creds(serverCredentials),
	)
	if err != nil {
		log.Fatalf("failed to create server: %v\n", err)
	}

	go func() {
		log.Fatal(server.Serve(lis))
	}()

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	log.Printf("received signal: %v\n", <-ch)
	clog.Close()
	server.GracefulStop()
	log.Println("server shutdown")
}
