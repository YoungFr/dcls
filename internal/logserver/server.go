package logserver

import (
	"context"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_auth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	api "github.com/youngfr/dcls/api/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// 配置实际使用了哪一种日志存储结构实现及哪一种访问控制实现
type LogImplConfig struct {
	CommitLog  CommitLog
	Authorizer Authorizer
}

var _ api.LogServer = (*gRPCServer)(nil)

type gRPCServer struct {
	*LogImplConfig
	api.UnimplementedLogServer
}

// 根据实际使用的日志存储结构、访问控制机制和服务器选项创建 gRPC 服务器
func NewgRPCServer(c *LogImplConfig, opts ...grpc.ServerOption) (*grpc.Server, error) {
	opts = append(opts, grpc.UnaryInterceptor(
		grpc_middleware.ChainUnaryServer(grpc_auth.UnaryServerInterceptor(authenticate)),
	))

	// 1. 调用 grpc.NewServer 方法
	s := grpc.NewServer(opts...)

	// 2. 创建自己的服务器
	srv := &gRPCServer{LogImplConfig: c}

	// 3. 调用 ProtoBuf 自动生成的注册方法
	api.RegisterLogServer(s, srv)

	return s, nil
}

var errNoAuthorizationUsed = status.New(codes.Unauthenticated, "no authorization being used").Err()

func (s *gRPCServer) Read(ctx context.Context, req *api.ReadRequest) (*api.ReadResponse, error) {
	if s.Authorizer == nil {
		return nil, errNoAuthorizationUsed
	}
	// 超级用户、普通用户和只读用户都可以读取日志
	if err := s.Authorizer.Authorize(subject(ctx), objects, readAction); err != nil {
		return nil, err
	}
	record, err := s.CommitLog.Read(req.Offset)
	if err != nil {
		return nil, err
	}
	return &api.ReadResponse{Record: record}, nil
}

func (s *gRPCServer) Append(ctx context.Context, req *api.AppendRequest) (*api.AppendResponse, error) {
	if s.Authorizer == nil {
		return nil, errNoAuthorizationUsed
	}
	// 超级用户和普通用户可以追加日志
	if err := s.Authorizer.Authorize(subject(ctx), objects, appendAction); err != nil {
		return nil, err
	}
	absOff, err := s.CommitLog.Append(req.Record)
	if err != nil {
		return nil, err
	}
	return &api.AppendResponse{Offset: absOff}, nil
}

const (
	RESET_SUCC = "Reset SUCCESS"
	RESET_FAIL = "Reset FAILED"
)

func (s *gRPCServer) Reset(ctx context.Context, req *api.ResetRequest) (*api.ResetResponse, error) {
	if s.Authorizer == nil {
		return nil, errNoAuthorizationUsed
	}
	// 只有超级用户可以删除所有日志
	if err := s.Authorizer.Authorize(subject(ctx), objects, resetAction); err != nil {
		return nil, err
	}
	if err := s.CommitLog.Reset(); err != nil {
		return &api.ResetResponse{Reply: RESET_FAIL}, err
	}
	return &api.ResetResponse{Reply: RESET_SUCC}, nil
}
