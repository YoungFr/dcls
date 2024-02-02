package main

import (
	"fmt"
	"log"
	"net"
	"os"

	dclslog "github.com/youngfr/dcls/internal/log"
	"github.com/youngfr/dcls/internal/logserver"
	"github.com/youngfr/dcls/internal/tlsconfig"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const logStoringDir = "log-services"

func main() {
	// 在 8080 端口监听
	lis, err := net.Listen("tcp", "127.0.0.1:8080")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	fmt.Println("Listening on port 8080...")
	defer lis.Close()

	// 创建存储日志的目录
	err = os.Mkdir(logStoringDir, 0744)
	if err != nil {
		log.Fatalf("failed to create log storing directory: %v", err)
	}

	// 使用默认配置创建一个日志存储结构
	clog, err := dclslog.NewLog(logStoringDir, dclslog.Config{})
	if err != nil {
		log.Fatalf("failed to create Log object: %v", err)
	}
	defer clog.Close()

	// 服务端双向 TLS 认证配置
	serverTLSConfig, err := tlsconfig.SetupTLSConfig(tlsconfig.TLSConfig{
		IsServerConfig:  true,
		EnableMutualTLS: true,
		CertFile:        tlsconfig.ServerCertFile,
		KeyFile:         tlsconfig.ServerKeyFile,
		CAFile:          tlsconfig.CAFile,
		ServerName:      lis.Addr().String(),
	})
	if err != nil {
		log.Fatalf("failed to set server mTLS config: %v", err)
	}
	serverCredentials := credentials.NewTLS(serverTLSConfig)

	// 创建 gRPC 服务器
	server, err := logserver.NewgRPCServer(&logserver.LogImplConfig{CommitLog: clog}, grpc.Creds(serverCredentials))
	if err != nil {
		log.Fatalf("failed to create server: %v", err)
	}

	if err = server.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
	defer server.Stop()
}
