package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/youngfr/dcls/internal/auth"
	dclslog "github.com/youngfr/dcls/internal/log"
	"github.com/youngfr/dcls/internal/logserver"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const logStoringDir = "log-services"

var port = flag.Int("port", 8080, "the port to serve on")

func main() {
	flag.Parse()

	lis, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", *port))
	if err != nil {
		log.Fatalf("failed to listen: %v\n", err)
	}
	log.Printf("server starting on port %d...\n", *port)

	// 创建存储日志的对象
	if _, err := os.Stat(logStoringDir); os.IsNotExist(err) {
		if err := os.Mkdir(logStoringDir, 0744); err != nil {
			log.Fatalf("failed to create log storing directory: %v\n", err)
		}
	}
	clog, err := dclslog.NewLog(logStoringDir, dclslog.Config{})
	if err != nil {
		log.Fatalf("failed to create Log object: %v\n", err)
	}

	// 双向 TLS 设置
	serverTLSConfig, err := auth.SetupTLSConfig(auth.TLSConfig{
		IsServerConfig:  true,
		EnableMutualTLS: true,
		CertFile:        auth.ServerCertFile,
		KeyFile:         auth.ServerKeyFile,
		CAFile:          auth.CAFile,
		ServerName:      lis.Addr().String(),
	})
	if err != nil {
		log.Fatalf("failed to setup server mTLS: %v\n", err)
	}
	serverCredentials := credentials.NewTLS(serverTLSConfig)

	// 创建服务器
	server, err := logserver.NewgRPCServer(
		&logserver.LogImplConfig{
			CommitLog:  clog,
			Authorizer: auth.NewAuthorizer(auth.ACLModelFile, auth.ACLPolicyFile),
		},
		grpc.Creds(serverCredentials),
	)
	if err != nil {
		log.Fatalf("failed to create server: %v\n", err)
	}

	go func() {
		log.Fatal(server.Serve(lis))
	}()

	// 优雅地关闭服务器
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	log.Printf("received signal: %v\n", <-ch)
	clog.Close()
	server.GracefulStop()
	log.Printf("server shutdown\n")
}
