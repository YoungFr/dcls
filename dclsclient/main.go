package main

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	api "github.com/youngfr/dcls/api/v1"
	"github.com/youngfr/dcls/internal/tlsconfig"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func main() {
	// 客户端双向 TLS 认证配置
	clientTLSConfig, err := tlsconfig.SetupTLSConfig(tlsconfig.TLSConfig{
		IsServerConfig:  false,
		EnableMutualTLS: true,
		CertFile:        tlsconfig.ClientCertFile,
		KeyFile:         tlsconfig.ClientKeyFile,
		CAFile:          tlsconfig.CAFile,
	})
	if err != nil {
		log.Fatalf("failed to set client mTLS config: %v", err)
	}
	clientCredentials := credentials.NewTLS(clientTLSConfig)
	clientOptions := []grpc.DialOption{grpc.WithTransportCredentials(clientCredentials)}

	// 连接到服务器
	conn, err := grpc.Dial("127.0.0.1:8080", clientOptions...)
	if err != nil {
		log.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	// 创建客户端
	client := api.NewLogClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for i := 0; i < 5; i++ {
		if rsp, err := client.Produce(ctx, &api.ProduceRequest{
			Record: &api.Record{
				Offset: uint64(i),
				Value:  []byte(strconv.Itoa(i)),
			},
		}); err == nil {
			fmt.Println(rsp.Offset)
		}
	}
}
