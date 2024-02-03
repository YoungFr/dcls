package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	dclslog "github.com/youngfr/dcls/internal/log"
	"github.com/youngfr/dcls/internal/logserver"
	"github.com/youngfr/dcls/internal/tlscfg"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const logStoringDir = "log-services"

func main() {
	// 在 8080 端口开启监听
	lis, err := net.Listen("tcp", "127.0.0.1:8080")
	if err != nil {
		log.Fatalf("failed to listen: %v\n", err)
	}
	fmt.Println("Listening on port 8080...")

	// 创建存储日志的目录
	_, err = os.Stat(logStoringDir)
	if os.IsNotExist(err) {
		if err = os.Mkdir(logStoringDir, 0744); err != nil {
			log.Fatalf("failed to create log storing directory: %v\n", err)
		}
	}

	// 创建一个日志存储结构
	// 这里我们以 1 作为日志的起始下标
	clog, err := dclslog.NewLog(logStoringDir, dclslog.Config{
		Segment: struct {
			MaxStoreBytes uint64
			MaxIndexBytes uint64
			InitialOffset uint64
		}{
			InitialOffset: 1,
		},
	})
	if err != nil {
		log.Fatalf("failed to create Log object: %v\n", err)
	}

	// 服务端双向 TLS 认证配置
	serverTLSConfig, err := tlscfg.SetupTLSConfig(tlscfg.TLSConfig{
		IsServerConfig:  true,
		EnableMutualTLS: true,
		CertFile:        tlscfg.ServerCertFile,
		KeyFile:         tlscfg.ServerKeyFile,
		CAFile:          tlscfg.CAFile,
		ServerName:      lis.Addr().String(),
	})
	if err != nil {
		log.Fatalf("failed to setup server mTLS config: %v\n", err)
	}
	serverCredentials := credentials.NewTLS(serverTLSConfig)

	// 创建 gRPC 服务器
	server, err := logserver.NewgRPCServer(
		&logserver.LogImplConfig{CommitLog: clog},
		grpc.Creds(serverCredentials),
	)
	if err != nil {
		log.Fatalf("failed to create server: %v\n", err)
	}

	// 启动服务
	go func() {
		if err = server.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v\n", err)
		}
	}()

	// 优雅关闭
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	log.Printf("received signal: %v\n", <-ch)
	clog.Close()
	server.GracefulStop()
	log.Println("server shutdown")
}
