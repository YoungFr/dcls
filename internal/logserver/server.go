package logserver

import (
	"context"

	api "github.com/youngfr/dcls/api/v1"
	"github.com/youngfr/dcls/internal/log"
	"google.golang.org/grpc"
)

// 这里的 CommitLog 是一个通用的日志存储结构需要实现的接口
// 这意味着我们在服务端真正使用的日志存储结构可以
// 不使用 internal/log 目录下的实现的 Log 结构体
// 而是只要实现这三种方法即可
type CommitLog interface {

	// 将一条日志追加到日志存储结构中
	// 成功时返回这条日志的下标
	Append(*api.Record) (uint64, error)

	// 给定一个下标读取对应的日志
	// 成功时返回读取到的日志记录
	Read(uint64) (*api.Record, error)

	// 删除当前日志存储结构中的所有日志
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
	// 创建 gRPC 服务器的 3 个标准步骤
	// 1
	s := grpc.NewServer(opts...)

	// 2
	srv, err := newgRPCServer(c)
	if err != nil {
		return nil, err
	}

	// 3
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
		return &api.ResetResponse{Ans: "Reset FAILED"}, err
	} else {
		return &api.ResetResponse{Ans: "Reset SUCCESS"}, nil
	}
}
