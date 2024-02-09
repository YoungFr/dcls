package tests

import (
	"context"
	"net"
	"os"
	"os/signal"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	api "github.com/youngfr/dcls/api/v1"
	"github.com/youngfr/dcls/internal/auth"
	dclslog "github.com/youngfr/dcls/internal/log"
	"github.com/youngfr/dcls/internal/logserver"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func TestAuthenticationAndAuthorization(t *testing.T) {
	t.Run("authentication and authorization test", func(t *testing.T) {
		// ------------------------------ Server ------------------------------

		// 服务端在 8080 端口开启监听
		lis, err := net.Listen("tcp", "127.0.0.1:8080")
		require.NoError(t, err)

		// 创建存储日志的目录
		ldir, err := os.MkdirTemp("", "server-log-services")
		require.NoError(t, err)

		// 日志的起始下标
		initialOffset := uint64(1)

		// 创建日志存储结构
		clog, err := dclslog.NewLog(ldir, dclslog.Config{
			Segment: struct {
				MaxStoreBytes uint64
				MaxIndexBytes uint64
				InitialOffset uint64
			}{
				InitialOffset: initialOffset,
			},
		})
		require.NoError(t, err)

		// 服务端采取双向 TLS 认证
		serverTLSConfig, err := auth.SetupTLSConfig(auth.TLSConfig{
			IsServerConfig:  true,
			EnableMutualTLS: true,
			CertFile:        auth.ServerCertFile,
			KeyFile:         auth.ServerKeyFile,
			CAFile:          auth.CAFile,
			ServerName:      lis.Addr().String(),
		})
		require.NoError(t, err)
		serverCredentials := credentials.NewTLS(serverTLSConfig)

		// 创建 gRPC 日志服务器
		server, err := logserver.NewgRPCServer(
			&logserver.LogImplConfig{
				CommitLog:  clog,
				Authorizer: auth.NewAuthorizer(auth.ACLModelFile, auth.ACLPolicyFile),
			},
			grpc.Creds(serverCredentials),
		)
		require.NoError(t, err)

		// 启动服务
		go func() {
			server.Serve(lis)
		}()
		time.Sleep(1 * time.Second)

		// 所有客户端操作完成后向这个管道发送一个信号来停止服务器的服务
		serverShutDown := make(chan os.Signal, 1)
		signal.Notify(serverShutDown, syscall.SIGINT)

		// 终止服务器后向这个管道发送数据来让整个测试线程停止
		servToTestChan := make(chan int, 1)

		// 优雅地停止服务
		go func() {
			<-serverShutDown
			defer func() {
				servToTestChan <- 1
			}()
			clog.Close()
			server.GracefulStop()
		}()

		// ------------------------------ Server ------------------------------

		// 测试使用的日志数据
		records := []*api.Record{
			{Value: []byte("The sun is for the day")},
			{Value: []byte("The moon is for the night")},
			{Value: []byte("And you forever")},
		}

		// --------------------------- Root Client ----------------------------

		// 超级用户采取双向 TLS 认证
		rootClientTLSConfig, err := auth.SetupTLSConfig(auth.TLSConfig{
			IsServerConfig:  false,
			EnableMutualTLS: true,
			CertFile:        auth.RootClientCertFile,
			KeyFile:         auth.RootClientKeyFile,
			CAFile:          auth.CAFile,
		})
		require.NoError(t, err)
		rootClientCredentials := credentials.NewTLS(rootClientTLSConfig)
		rootClientOptions := []grpc.DialOption{
			grpc.WithTransportCredentials(rootClientCredentials),
		}

		// 超级用户连接到服务端
		rootConn, err := grpc.Dial("127.0.0.1:8080", rootClientOptions...)
		require.NoError(t, err)

		// 测试线程 -> 超级用户线程 -> 普通用户线程 -> 只读用户线程 -> 服务器终止线程 -> 测试线程
		testToRootChan := make(chan int, 1)
		rootToOrdiChan := make(chan int, 1)
		ordiToReadChan := make(chan int, 1)

		go func() {
			testToRootChan <- 1
		}()

		// 测试超级用户客户端
		go func() {
			<-testToRootChan
			defer func() {
				rootConn.Close()
				// 启动普通用户客户端
				rootToOrdiChan <- 1
			}()

			// 创建超级用户客户端
			rootClient := api.NewLogClient(rootConn)
			ctx := context.Background()

			// 超级用户有读取日志的权限
			// 在读取第一条日志时返回空值和错误
			readRsp, err := rootClient.Read(ctx, &api.ReadRequest{Offset: 1})
			require.Empty(t, readRsp)
			require.Error(t, err)

			// 超级用户有追加日志的权限
			// 追加日志成功后返回正确的下标
			for i, record := range records {
				appendRsp, err := rootClient.Append(ctx, &api.AppendRequest{Record: record})
				require.NoError(t, err)
				require.Equal(t, initialOffset+uint64(i), appendRsp.Offset)
			}

			// 超级用户按照从新到老的顺序读取所有日志
			// 读取日志成功后返回正确的下标和日志内容
			for i := len(records) - 1; i >= 0; i-- {
				readOffset := initialOffset + uint64(i)
				readRsp, err = rootClient.Read(ctx, &api.ReadRequest{Offset: readOffset})
				require.NoError(t, err)
				require.Equal(t, readOffset, readRsp.Record.Offset)
				require.Equal(t, records[i].Value, readRsp.Record.Value)
			}

			// 如果读取日志时给定的下标超出范围则返回错误
			readRsp, err = rootClient.Read(ctx, &api.ReadRequest{Offset: initialOffset - 1})
			require.Empty(t, readRsp)
			require.Error(t, err)
			readRsp, err = rootClient.Read(ctx, &api.ReadRequest{Offset: uint64(len(records) + 1)})
			require.Empty(t, readRsp)
			require.Error(t, err)

			// 超级用户有清空所有日志的权限
			resetRsp, err := rootClient.Reset(ctx, &api.ResetRequest{})
			require.NoError(t, err)
			require.Equal(t, logserver.RESET_SUCC, resetRsp.Reply)

			// 在清空所有日志后再次读取会报错
			readRsp, err = rootClient.Read(ctx, &api.ReadRequest{Offset: 1})
			require.Empty(t, readRsp)
			require.Error(t, err)
		}()

		// --------------------------- Root Client ----------------------------

		// ------------------------- Ordinary Client --------------------------

		// 普通用户采取双向 TLS 认证
		ordinaryClientTLSConfig, err := auth.SetupTLSConfig(auth.TLSConfig{
			IsServerConfig:  false,
			EnableMutualTLS: true,
			CertFile:        auth.OrdinaryClientCertFile,
			KeyFile:         auth.OrdinaryClientKeyFile,
			CAFile:          auth.CAFile,
		})
		require.NoError(t, err)
		ordinaryClientCredentials := credentials.NewTLS(ordinaryClientTLSConfig)
		ordinaryClientOptions := []grpc.DialOption{
			grpc.WithTransportCredentials(ordinaryClientCredentials),
		}

		ordinaryConn, err := grpc.Dial("127.0.0.1:8080", ordinaryClientOptions...)
		require.NoError(t, err)

		go func() {
			<-rootToOrdiChan
			defer func() {
				ordinaryConn.Close()
				// 启动只读用户客户端
				ordiToReadChan <- 1
			}()

			// 创建普通用户客户端
			ordinaryClient := api.NewLogClient(ordinaryConn)
			ctx := context.Background()

			// 普通用户有读取日志的权限
			// 由于超级用户已经将日志清空
			// 所以读取第一条日志时会报错
			readRsp, err := ordinaryClient.Read(ctx, &api.ReadRequest{Offset: initialOffset})
			require.Empty(t, readRsp)
			require.Error(t, err)

			// 普通用户有追加日志的权限
			appendRsp, err := ordinaryClient.Append(ctx, &api.AppendRequest{Record: records[0]})
			require.NoError(t, err)
			require.Equal(t, initialOffset+uint64(0), appendRsp.Offset)

			// 普通用户有读取日志的权限
			readOffset := initialOffset + uint64(0)
			readRsp, err = ordinaryClient.Read(ctx, &api.ReadRequest{Offset: readOffset})
			require.NoError(t, err)
			require.Equal(t, readOffset, readRsp.Record.Offset)
			require.Equal(t, records[0].Value, readRsp.Record.Value)

			// 普通用户没有清空日志的权限
			// 在尝试清空日志时会返回错误
			resetRsp, err := ordinaryClient.Reset(ctx, &api.ResetRequest{})
			require.Empty(t, resetRsp)
			require.Error(t, err)
		}()

		// ------------------------- Ordinary Client --------------------------

		// ------------------------- ReadOnly Client --------------------------

		// 只读用户采取双向 TLS 认证
		readOnlyClientTLSConfig, err := auth.SetupTLSConfig(auth.TLSConfig{
			IsServerConfig:  false,
			EnableMutualTLS: true,
			CertFile:        auth.ReadOnlyClientCertFile,
			KeyFile:         auth.ReadOnlyClientKeyFile,
			CAFile:          auth.CAFile,
		})
		require.NoError(t, err)
		readOnlyClientCredentials := credentials.NewTLS(readOnlyClientTLSConfig)
		readOnlyClientOptions := []grpc.DialOption{
			grpc.WithTransportCredentials(readOnlyClientCredentials),
		}

		readOnlyConn, err := grpc.Dial("127.0.0.1:8080", readOnlyClientOptions...)
		require.NoError(t, err)

		go func() {
			<-ordiToReadChan
			defer func() {
				readOnlyConn.Close()
				// 终止服务器服务
				serverShutDown <- syscall.SIGINT
			}()

			readOnlyClient := api.NewLogClient(readOnlyConn)
			ctx := context.Background()

			// 只读用户只有读取日志的权限
			// 之前普通用户追加了一条日志
			// 所以只读用户可以直接读取它
			readOffset := initialOffset + uint64(0)
			readRsp, err := readOnlyClient.Read(ctx, &api.ReadRequest{Offset: readOffset})
			require.NoError(t, err)
			require.Equal(t, readOffset, readRsp.Record.Offset)
			require.Equal(t, records[0].Value, readRsp.Record.Value)

			// 只读用户尝试追加日志时报错
			appendRsp, err := readOnlyClient.Append(ctx, &api.AppendRequest{Record: records[1]})
			require.Empty(t, appendRsp)
			require.Error(t, err)

			// 只读用户尝试清空日志时报错
			resetRsp, err := readOnlyClient.Reset(ctx, &api.ResetRequest{})
			require.Empty(t, resetRsp)
			require.Error(t, err)
		}()

		// ------------------------- ReadOnly Client --------------------------

		// 等待服务器终止
		<-servToTestChan
	})
}
