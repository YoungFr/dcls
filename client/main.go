package main

import (
	"context"
	"fmt"
	"log"

	api "github.com/youngfr/dcls/api/v1"
	"github.com/youngfr/dcls/internal/auth"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func main() {
	rootClientTLSConfig, err := auth.SetupTLSConfig(auth.TLSConfig{
		IsServerConfig:  false,
		EnableMutualTLS: true,
		CertFile:        auth.RootClientCertFile,
		KeyFile:         auth.RootClientKeyFile,
		CAFile:          auth.CAFile,
	})
	if err != nil {
		log.Fatalf("failed to setup root client mTLS: %v\n", err)
	}
	rootClientCredentials := credentials.NewTLS(rootClientTLSConfig)
	rootClientOptions := []grpc.DialOption{
		grpc.WithTransportCredentials(rootClientCredentials),
	}

	rootConn, err := grpc.Dial("127.0.0.1:8080", rootClientOptions...)
	if err != nil {
		log.Fatalf("failed to connect: %v\n", err)
	}

	rootClient := api.NewLogClient(rootConn)
	ctx := context.Background()

	appendRsp, err := rootClient.Append(ctx, &api.AppendRequest{
		Record: &api.Record{
			Value: []byte("my first log"),
		},
	})
	if err != nil {
		log.Fatalf("append failed: %v\n", err)
	}
	fmt.Printf("new appended log offset: %d\n", appendRsp.Offset)

	readRsp, err := rootClient.Read(ctx, &api.ReadRequest{Offset: uint64(0)})
	if err != nil {
		log.Fatalf("read failed: %v\n", err)
	}
	fmt.Printf("LOGS[%d] contents: %s\n", 0, readRsp.Record.Value)
}
