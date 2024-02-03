package main

import (
	"context"
	"fmt"
	"log"
	"time"

	api "github.com/youngfr/dcls/api/v1"
	"github.com/youngfr/dcls/internal/tlscfg"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func main() {
	// 客户端双向 TLS 认证配置
	clientTLSConfig, err := tlscfg.SetupTLSConfig(tlscfg.TLSConfig{
		IsServerConfig:  false,
		EnableMutualTLS: true,
		CertFile:        tlscfg.ClientCertFile,
		KeyFile:         tlscfg.ClientKeyFile,
		CAFile:          tlscfg.CAFile,
	})
	if err != nil {
		log.Fatalf("failed to setup client mTLS config: %v\n", err)
	}
	clientCredentials := credentials.NewTLS(clientTLSConfig)
	clientOptions := []grpc.DialOption{
		grpc.WithTransportCredentials(clientCredentials),
	}

	// 连接到服务器
	conn, err := grpc.Dial("127.0.0.1:8080", clientOptions...)
	if err != nil {
		log.Fatalf("failed to connect: %v\n", err)
	}
	defer conn.Close()

	// 创建客户端
	client := api.NewLogClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 要写入的日志
	records := []*api.Record{
		{Value: []byte("The sun is for the day")},
		{Value: []byte("The moon is for the night")},
		{Value: []byte("And you forever")},
	}

	// 读取第一条日志时报错
	fmt.Println("-------- Read Error --------")
	_, err = client.Read(ctx, &api.ReadRequest{Offset: 1})
	if err == nil {
		log.Fatalf("unexpected result: error should be non-nil\n")
	}
	fmt.Printf("failed to read: %v\n", err)

	// 追加若干条日志
	fmt.Println("---------- Append ----------")
	for _, record := range records {
		appendRsp, err := client.Append(ctx, &api.AppendRequest{Record: record})
		if err != nil {
			log.Fatalf("failed to append: %v\n", err)
		} else {
			fmt.Printf("log content: [%s] offset: [%d]\n",
				record.Value,
				appendRsp.Offset,
			)
		}
	}

	// 按照从新到老的顺序读取所有日志
	fmt.Println("----------- Read -----------")
	for i := len(records); i >= 1; i-- {
		readRsp, err := client.Read(ctx, &api.ReadRequest{Offset: uint64(i)})
		if err != nil {
			log.Fatalf("failed to read: %v\n", err)
		} else {
			fmt.Printf("offset: [%d] log content: [%s]\n",
				readRsp.Record.Offset,
				readRsp.Record.Value,
			)
		}
	}

	// 读取时给定的下标超出范围报错
	fmt.Println("-------- Read Error --------")
	_, err = client.Read(ctx, &api.ReadRequest{Offset: 0})
	if err == nil {
		log.Fatalf("unexpected result: error should be non-nil\n")
	}
	fmt.Printf("failed to read: %v\n", err)
	_, err = client.Read(ctx, &api.ReadRequest{Offset: uint64(len(records) + 1)})
	if err == nil {
		log.Fatalf("unexpected result: error should be non-nil\n")
	}
	fmt.Printf("failed to read: %v\n", err)

	// 清空所有日志
	fmt.Println("------- Reset Result -------")
	resetRsp, err := client.Reset(ctx, &api.ResetRequest{})
	if err != nil {
		log.Fatalf("failed to reset: %v\n", err)
	}
	fmt.Println(resetRsp.Reply)

	// 再次读取第一条日志时报错
	fmt.Println("----- Read After Reset -----")
	_, err = client.Read(ctx, &api.ReadRequest{Offset: 1})
	if err == nil {
		log.Fatalf("unexpected result: error should be non-nil\n")
	}
	fmt.Printf("failed to read after reset: %v\n", err)

	fmt.Println("PASS")
}
