package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	api "github.com/youngfr/dcls/api/v1"
	"github.com/youngfr/dcls/internal/auth"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var addr = flag.String("addr", "127.0.0.1:8080", "the address to connect to")

func main() {
	flag.Parse()

	// 双向 TLS 设置
	clientTLSConfig, err := auth.SetupTLSConfig(auth.TLSConfig{
		IsServerConfig:  false,
		EnableMutualTLS: true,
		CertFile:        auth.RootClientCertFile,
		KeyFile:         auth.RootClientKeyFile,
		CAFile:          auth.CAFile,
	})
	if err != nil {
		log.Fatalf("failed to setup client mTLS: %v\n", err)
	}
	clientOptions := []grpc.DialOption{
		grpc.WithTransportCredentials(credentials.NewTLS(clientTLSConfig)),
	}

	conn, err := grpc.Dial(*addr, clientOptions...)
	if err != nil {
		log.Fatalf("failed to connect: %v\n", err)
	}

	client := api.NewLogClient(conn)
	ctx := context.Background()

	sc := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("> ")
		if sc.Scan() {
			args := strings.Fields(sc.Text())
			if len(args) <= 2 {
				switch strings.ToLower(args[0]) {
				case "append":
					if appendRsp, err := client.Append(ctx, &api.AppendRequest{
						Record: &api.Record{
							Value: []byte(args[1]),
						},
					}); err != nil {
						fmt.Printf("append failed: %v\n", err)
					} else {
						fmt.Printf("offset: %d\n", appendRsp.Offset)
					}
				case "read":
					offset, err := strconv.Atoi(args[1])
					if err != nil {
						fmt.Printf("parse read offset failed: %v\n", err)
					} else {
						if readRsp, err := client.Read(ctx, &api.ReadRequest{
							Offset: uint64(offset),
						}); err != nil {
							fmt.Printf("read failed: %v\n", err)
						} else {
							fmt.Printf("%s\n", readRsp.Record.Value)
						}
					}
				case "reset":
					if resetRsp, err := client.Reset(ctx, &api.ResetRequest{}); err != nil {
						fmt.Printf("reset failed: %v\n", err)
					} else {
						fmt.Printf("%s\n", resetRsp.Reply)
					}
				case "q", "quit":
					return
				default:
					fmt.Printf("unknown command\n")
				}
			} else {
				fmt.Printf("too many params: %d\n", len(args))
			}
		}
	}
}
