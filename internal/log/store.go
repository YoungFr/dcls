package log

import (
	"bufio"
	"encoding/binary"
	"os"
	"sync"
)

type store struct {
	mu sync.Mutex

	// 存储日志记录的文件
	*os.File

	// 写缓冲区
	buf *bufio.Writer

	// 文件大小
	size uint64
}

func newStore(f *os.File) (*store, error) {
	finfo, err := os.Stat(f.Name())
	if err != nil {
		return nil, err
	}

	// 获取文件当前的大小，单位是字节
	size := uint64(finfo.Size())

	return &store{
		File: f,
		// 缓冲区的默认大小是 4096 字节
		buf:  bufio.NewWriter(f),
		size: size,
	}, nil
}

// 将数字写入文件时使用的字节序
var order = binary.BigEndian

// 表示一条记录的长度的数字所占用的字节数
const lenSize = 8 // sizeof(uint64)

// 将一条记录追加写入文件的末尾
// 写入时会先写入记录的长度再写入记录的内容
// 这样在后续读取时就能知道应该读出多少字节
//
// 返回值 n 表示实际写入的字节数
// 返回值 pos 表示该条记录是从文件的第几个字节开始存储的
func (s *store) Append(b []byte) (n uint64, pos uint64, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 只支持追加写入
	// 所以新写入记录的起始索引就是写入前文件的大小
	pos = s.size

	// 使用缓冲 I/O 而不是直接写到文件中
	// 可以减少系统调用的次数从而提高性能
	if err := binary.Write(s.buf, order, uint64(len(b))); err != nil {
		return 0, 0, err
	}
	w, err := s.buf.Write(b)
	if err != nil {
		return 0, 0, err
	}

	// 实际写入了 w+lenSize 个字节
	w += lenSize
	s.size += uint64(w)

	return uint64(w), pos, nil
}

// 读出从文件的第 pos 个字节开始的那条记录
func (s *store) Read(pos uint64) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 首先保证将缓冲区中的内容写到文件中
	if err := s.buf.Flush(); err != nil {
		return nil, err
	}

	// 先读出记录的长度
	size := make([]byte, lenSize)
	if _, err := s.File.ReadAt(size, int64(pos)); err != nil {
		return nil, err
	}

	// 再读出由长度指定的字节数即是记录的内容
	b := make([]byte, order.Uint64(size))
	if _, err := s.File.ReadAt(b, int64(pos+lenSize)); err != nil {
		return nil, err
	}

	return b, nil
}

// 将存储的日志写入磁盘并关闭相应的文件
func (s *store) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.buf.Flush(); err != nil {
		return err
	}
	return s.File.Close()
}
