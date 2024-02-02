package server

import (
	"context"

	api "github.com/youngfr/dcls/api/v1"
	"github.com/youngfr/dcls/internal/log"
	"google.golang.org/grpc"
)

// 这里的 CommitLog 是一个通用的日志存储结构需要实现的接口
//
// 这意味着我们在服务端真正使用的日志存储结构可以
// 不使用 internal/log 目录下的实现的 Log 结构体
//
// 但是都必须实现 Append (追加一条记录) 和 Read (读取一条记录) 方法
type CommitLog interface {
	Append(*api.Record) (uint64, error)
	Read(uint64) (*api.Record, error)
	Reset() error
}

// 保证 *log.Log 实现了 CommitLog 接口
var _ CommitLog = (*log.Log)(nil)

// 配置实际使用了哪一种日志存储结构实现
// 假如以后我们不使用 *log.Log 而改用了其他实现来存储日志
// 我们只需要修改这里的配置即可
type LogImplConfig struct {
	CommitLog CommitLog
}

// 根据实际使用的日志存储结构和服务器选项创建 gRPC 服务器
func NewgRPCServer(c *LogImplConfig, opts ...grpc.ServerOption) (*grpc.Server, error) {
	s := grpc.NewServer(opts...)
	srv, err := newgRPCServer(c)
	if err != nil {
		return nil, err
	}
	api.RegisterLogServer(s, srv)
	return s, nil
}

func newgRPCServer(c *LogImplConfig) (*gRPCServer, error) {
	return &gRPCServer{LogImplConfig: c}, nil
}

// 我们的服务器需要实现 api.LogServer 接口
var _ api.LogServer = (*gRPCServer)(nil)

type gRPCServer struct {
	// 日志存储结构
	*LogImplConfig

	// 所有服务器实现都必须内嵌 UnimplementedLogServer 来保证向前兼容性
	api.UnimplementedLogServer
}

// 追加一条日志
func (s *gRPCServer) Produce(ctx context.Context, req *api.ProduceRequest) (*api.ProduceResponse, error) {
	absOff, err := s.CommitLog.Append(req.Record)
	if err != nil {
		return nil, err
	}
	return &api.ProduceResponse{Offset: absOff}, nil
}

// 读取一条日志
func (s *gRPCServer) Consume(ctx context.Context, req *api.ConsumeRequest) (*api.ConsumeResponse, error) {
	record, err := s.CommitLog.Read(req.Offset)
	if err != nil {
		return nil, err
	}
	return &api.ConsumeResponse{Record: record}, nil
}

// 删除所有日志
func (s *gRPCServer) Reset(ctx context.Context, req *api.ResetRequest) (*api.ResetResponse, error) {
	if err := s.CommitLog.Reset(); err != nil {
		return &api.ResetResponse{Ans: "Reset Failed"}, err
	} else {
		return &api.ResetResponse{Ans: "Reset OK"}, nil
	}
}
