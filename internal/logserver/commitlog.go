package logserver

import (
	api "github.com/youngfr/dcls/api/v1"
	"github.com/youngfr/dcls/internal/log"
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

// 在 log 包中的 *log.Log 实现了 CommitLog 接口
var _ CommitLog = (*log.Log)(nil)
