package logserver

import (
	"context"
	"net"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	api "github.com/youngfr/dcls/api/v1"
	"github.com/youngfr/dcls/internal/log"
	"github.com/youngfr/dcls/internal/tlscfg"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"
)

func TestServer(t *testing.T) {
	testcases := map[string]func(t *testing.T, client api.LogClient, config *LogImplConfig){
		"produce/consume a message to/from the log succeeeds": testProduceConsume,
		"consume past log boundary fails":                     testConsumePastBoundary,
	}
	for scenario, fn := range testcases {
		t.Run(scenario, func(t *testing.T) {
			client, config, teardown := setupTest(t, nil)
			defer teardown()
			fn(t, client, config)
		})
	}
}

// setup
func setupTest(t *testing.T, fn func(*LogImplConfig)) (client api.LogClient, config *LogImplConfig, teardown func()) {
	t.Helper()

	// 双向 TLS 认证测试

	// ---------- 客户端 ----------
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	// 客户端双向 TLS 认证需要根证书、客户端证书和客户端私钥
	clientTLSConfig, err := tlscfg.SetupTLSConfig(tlscfg.TLSConfig{
		IsServerConfig:  false,
		EnableMutualTLS: true,
		CertFile:        tlscfg.ClientCertFile,
		KeyFile:         tlscfg.ClientKeyFile,
		CAFile:          tlscfg.CAFile,
	})
	require.NoError(t, err)

	// 客户端连接选项
	clientCredentials := credentials.NewTLS(clientTLSConfig)
	clientOptions := []grpc.DialOption{
		grpc.WithTransportCredentials(clientCredentials),
	}

	conn, err := grpc.Dial(ln.Addr().String(), clientOptions...)
	require.NoError(t, err)
	// ---------- 客户端 ----------

	// ---------- 服务端 ----------
	// 服务端双向 TLS 认证需要根证书、服务端证书和服务端私钥
	serverTLSConfig, err := tlscfg.SetupTLSConfig(tlscfg.TLSConfig{
		IsServerConfig:  true,
		EnableMutualTLS: true,
		CertFile:        tlscfg.ServerCertFile,
		KeyFile:         tlscfg.ServerKeyFile,
		CAFile:          tlscfg.CAFile,
		ServerName:      ln.Addr().String(),
	})
	require.NoError(t, err)

	serverCredentials := credentials.NewTLS(serverTLSConfig)

	// 日志存储目录
	dir, err := os.MkdirTemp("", "server-test")
	require.NoError(t, err)

	// 新建 Log 对象
	clog, err := log.NewLog(dir, log.Config{})
	require.NoError(t, err)

	config = &LogImplConfig{
		CommitLog: clog,
	}
	if fn != nil {
		fn(config)
	}

	server, err := NewgRPCServer(config, grpc.Creds(serverCredentials))
	require.NoError(t, err)
	// ---------- 服务端 ----------

	go func() {
		server.Serve(ln)
	}()

	client = api.NewLogClient(conn)

	return client, config, func() {
		server.Stop()
		conn.Close()
		ln.Close()
		clog.Close()
	}
}

// produceconsume
func testProduceConsume(t *testing.T, client api.LogClient, config *LogImplConfig) {
	ctx := context.Background()

	want := &api.Record{Value: []byte("hello world")}

	// append
	produce, err := client.Append(ctx, &api.AppendRequest{Record: want})
	require.NoError(t, err)

	// read
	consume, err := client.Read(ctx, &api.ReadRequest{Offset: produce.Offset})
	require.NoError(t, err)

	require.Equal(t, want.Value, consume.Record.Value)
	require.Equal(t, want.Offset, consume.Record.Offset)
}

// consumeerror
func testConsumePastBoundary(t *testing.T, client api.LogClient, config *LogImplConfig) {
	ctx := context.Background()

	produce, err := client.Append(
		ctx,
		&api.AppendRequest{
			Record: &api.Record{Value: []byte("hello world")},
		},
	)
	require.NoError(t, err)

	consume, err := client.Read(ctx, &api.ReadRequest{
		Offset: produce.Offset + 1,
	})
	if consume != nil {
		t.Fatal("consume not nil")
	}

	got := status.Code(err)
	want := status.Code(api.ErrOffsetOutOfRange{}.GRPCStatus().Err())
	if got != want {
		t.Fatalf("got err: %v, want: %v", got, want)
	}
}
