package logserver

import (
	"context"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_auth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	api "github.com/youngfr/dcls/api/v1"
	"google.golang.org/grpc"
)

// 配置实际使用了哪一种日志存储结构实现及哪一种 ACL 访问控制实现
type LogImplConfig struct {
	CommitLog  CommitLog
	Authorizer Authorizer
}

type gRPCServer struct {
	*LogImplConfig
	api.UnimplementedLogServer
}

var _ api.LogServer = (*gRPCServer)(nil)

func newgRPCServer(c *LogImplConfig) (*gRPCServer, error) {
	return &gRPCServer{LogImplConfig: c}, nil
}

// 根据实际使用的日志存储结构、访问控制机制和服务器选项创建 gRPC 服务器
func NewgRPCServer(c *LogImplConfig, opts ...grpc.ServerOption) (*grpc.Server, error) {
	opts = append(opts, grpc.UnaryInterceptor(
		grpc_middleware.ChainUnaryServer(grpc_auth.UnaryServerInterceptor(authenticate)),
	))

	// 1. 调用 grpc.NewServer 方法
	s := grpc.NewServer(opts...)

	// 2. 创建自己的服务器
	srv, err := newgRPCServer(c)
	if err != nil {
		return nil, err
	}

	// 3. 调用 PeotoBuf 自动生成的注册方法
	api.RegisterLogServer(s, srv)

	return s, nil
}

func (s *gRPCServer) Read(ctx context.Context, req *api.ReadRequest) (*api.ReadResponse, error) {
	// 超级用户、普通用户和只读用户都可以读取日志
	if err := s.Authorizer.Authorize(subject(ctx), objectWildcard, readAction); err != nil {
		return nil, err
	}
	record, err := s.CommitLog.Read(req.Offset)
	if err != nil {
		return nil, err
	}
	return &api.ReadResponse{Record: record}, nil
}

func (s *gRPCServer) Append(ctx context.Context, req *api.AppendRequest) (*api.AppendResponse, error) {
	// 超级用户和普通用户可以追加日志
	if err := s.Authorizer.Authorize(subject(ctx), objectWildcard, appendAction); err != nil {
		return nil, err
	}
	absOff, err := s.CommitLog.Append(req.Record)
	if err != nil {
		return nil, err
	}
	return &api.AppendResponse{Offset: absOff}, nil
}

func (s *gRPCServer) Reset(ctx context.Context, req *api.ResetRequest) (*api.ResetResponse, error) {
	// 只有超级用户可以删除所有日志
	if err := s.Authorizer.Authorize(subject(ctx), objectWildcard, resetAction); err != nil {
		return nil, err
	}
	if err := s.CommitLog.Reset(); err != nil {
		return &api.ResetResponse{Reply: "Reset FAILED!"}, err
	}
	return &api.ResetResponse{Reply: "Reset SUCCESS"}, nil
}
