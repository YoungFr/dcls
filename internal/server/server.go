package server

import (
	"context"

	api "github.com/youngfr/dcls/api/v1"
	"github.com/youngfr/dcls/internal/log"
	"google.golang.org/grpc"
)

// 通用的日志存储需要实现的接口
// 具体的日志存储结构可以不使用 *log.Log 中的实现
// 但是都必须实现 Append 和 Read 方法
type CommitLog interface {
	Append(*api.Record) (uint64, error)
	Read(uint64) (*api.Record, error)
}

var _ CommitLog = (*log.Log)(nil)

// 配置实际使用的日志存储实现方式
type Config struct {
	CommitLog CommitLog
}

type gRPCServer struct {
	// 所有服务器实现都必须内嵌 UnimplementedLogServer 来保证向前兼容性
	api.UnimplementedLogServer

	// 内嵌一个 *log.Log 对象
	*Config
}

// 我们的服务器需要实现 api.LogServer 接口
var _ api.LogServer = (*gRPCServer)(nil)

func NewgRPCServer(c *Config, opts ...grpc.ServerOption) (*grpc.Server, error) {
	s := grpc.NewServer(opts...)
	srv, err := newgRPCServer(c)
	if err != nil {
		return nil, err
	}
	api.RegisterLogServer(s, srv)
	return s, nil
}

func newgRPCServer(c *Config) (rpcServ *gRPCServer, err error) {
	rpcServ = &gRPCServer{
		Config: c,
	}
	return rpcServ, nil
}

func (s *gRPCServer) Produce(ctx context.Context, req *api.ProduceRequest) (*api.ProduceResponse, error) {
	absOff, err := s.CommitLog.Append(req.Record)
	if err != nil {
		return nil, err
	}
	return &api.ProduceResponse{Offset: absOff}, nil
}

func (s *gRPCServer) Consume(ctx context.Context, req *api.ConsumeRequest) (*api.ConsumeResponse, error) {
	record, err := s.CommitLog.Read(req.Offset)
	if err != nil {
		return nil, err
	}
	return &api.ConsumeResponse{Record: record}, nil
}

func (s *gRPCServer) ProduceStream(stream api.Log_ProduceStreamServer) error {
	for {
		req, err := stream.Recv()
		if err != nil {
			return err
		}
		rsp, err := s.Produce(stream.Context(), req)
		if err != nil {
			return err
		}
		if err = stream.Send(rsp); err != nil {
			return err
		}
	}
}

func (s *gRPCServer) ConsumeStream(req *api.ConsumeRequest, stream api.Log_ConsumeStreamServer) error {
	for {
		select {
		case <-stream.Context().Done():
			return nil
		default:
			rsp, err := s.Consume(stream.Context(), req)
			switch err.(type) {
			case nil:
				// do nothing
			case api.ErrOffsetOutOfRange:
				continue
			default:
				return err
			}
			if err = stream.Send(rsp); err != nil {
				return err
			}
			req.Offset++
		}
	}
}
