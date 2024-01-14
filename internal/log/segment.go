package log

import (
	"fmt"
	"os"
	"path"

	api "github.com/youngfr/dcls/api/v1"
	"google.golang.org/protobuf/proto"
)

type segment struct {
	// 一个 segment 包含存储和索引
	store *store
	index *index

	// 第一个索引项表示的记录的绝对下标
	baseOffset uint64
	// 下一条要写入的记录的绝对下标
	nextOffset uint64

	config Config
}

func newSegment(dir string, baseOffset uint64, c Config) (s *segment, err error) {
	s = &segment{
		baseOffset: baseOffset,
		config:     c,
	}

	// 创建存储文件
	storeFile, err := os.OpenFile(
		path.Join(dir, fmt.Sprintf("%d%s", baseOffset, ".store")),
		os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}
	if s.store, err = newStore(storeFile); err != nil {
		return nil, err
	}

	// 创建索引文件
	indexFile, err := os.OpenFile(
		path.Join(dir, fmt.Sprintf("%d%s", baseOffset, ".index")),
		os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return nil, err
	}
	if s.index, err = newIndex(indexFile, c); err != nil {
		return nil, err
	}

	// 设置 nextOffset 的值
	// 如果索引文件为空则 nextOffset 和 baseOffset 相等
	// 否则是 baseOffset 加上当前索引中的索引项数目
	if currMaxRelOff, _, err := s.index.Read(-1); err == errEmptyIndexFile {
		s.nextOffset = baseOffset
	} else {
		s.nextOffset = baseOffset + uint64(currMaxRelOff) + 1
	}

	return s, nil
}

func (s *segment) Append(record *api.Record) (offset uint64, err error) {
	// 当前记录的绝对下标
	curr := s.nextOffset
	record.Offset = curr

	// 序列化
	b, err := proto.Marshal(record)
	if err != nil {
		return 0, err
	}

	// 写入存储文件
	n, pos, err := s.store.Append(b)
	if n != uint64(len(b)+lenSize) || err != nil {
		return 0, err
	}

	// 将相对下标和它在存储文件中的位置写入索引文件
	if err := s.index.Write(uint32(s.nextOffset-s.baseOffset), pos); err != nil {
		return 0, err
	}

	s.nextOffset++
	return curr, nil
}

func (s *segment) Read(absOff uint64) (record *api.Record, err error) {
	// 先读取索引文件获取它在存储文件中的位置
	_, pos, err := s.index.Read(int64(absOff - s.baseOffset))
	if err != nil {
		return nil, err
	}

	// 从存储文件中读取数据
	b, err := s.store.Read(pos)
	if err != nil {
		return nil, err
	}

	// 反序列化
	record = &api.Record{}
	if err = proto.Unmarshal(b, record); err != nil {
		return nil, err
	}

	return record, nil
}

func (s *segment) Close() error {
	if err := s.index.Close(); err != nil {
		return err
	}
	if err := s.store.Close(); err != nil {
		return err
	}
	return nil
}

func (s *segment) IsMaxed() bool {
	return s.store.size >= s.config.Segment.MaxStoreBytes ||
		s.index.size >= s.config.Segment.MaxIndexBytes
}

func (s *segment) Remove() error {
	if err := s.Close(); err != nil {
		return err
	}
	if err := os.Remove(s.index.Name()); err != nil {
		return err
	}
	if err := os.Remove(s.store.Name()); err != nil {
		return err
	}
	return nil
}
