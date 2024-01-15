package log

import (
	"fmt"
	"os"
	"path"

	api "github.com/youngfr/dcls/api/v1"
	"google.golang.org/protobuf/proto"
)

type segment struct {
	// 每个 segment 都包含存储和索引
	store *store
	index *index

	// 本 segment 中存储的第一条记录的绝对下标
	baseAbsOffset uint64

	// 下一条要存储的记录的绝对下标
	nextAbsOffset uint64

	config Config
}

func newSegment(dir string, baseAbsOffset uint64, c Config) (s *segment, err error) {
	s = &segment{
		baseAbsOffset: baseAbsOffset,
		config:        c,
	}

	// 创建存储文件
	storeFile, err := os.OpenFile(
		path.Join(dir, fmt.Sprintf("%d%s", baseAbsOffset, ".store")),
		os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644,
	)
	if err != nil {
		return nil, err
	}
	if s.store, err = newStore(storeFile); err != nil {
		return nil, err
	}

	// 创建索引文件
	indexFile, err := os.OpenFile(
		path.Join(dir, fmt.Sprintf("%d%s", baseAbsOffset, ".index")),
		os.O_RDWR|os.O_CREATE, 0644,
	)
	if err != nil {
		return nil, err
	}
	if s.index, err = newIndex(indexFile, c); err != nil {
		return nil, err
	}

	// 设置新建 segment 时 nextAbsOffset 的值
	// 如果索引文件为空则下一条要存储的记录的绝对下标就是 baseAbsOffset
	//
	// 否则下一条要存储的记录的绝对下标是 baseAbsOffset 加上当前的索引项总数
	// 参考 (*index).Read 方法中的注释
	if currMaxRelOff, _, err := s.index.Read(-1); err == errEmptyIndexFile {
		s.nextAbsOffset = baseAbsOffset
	} else {
		s.nextAbsOffset = baseAbsOffset + (uint64(currMaxRelOff) + 1)
	}

	return s, nil
}

func (s *segment) Append(record *api.Record) (absOff uint64, err error) {
	// 只有在 index 还有空间时才会向存储文件和索引文件中写入内容
	if !s.index.HasSpace() {
		return 0, errNotEnoughIndexSpace
	}

	// 新写入的记录的绝对下标
	record.Offset = s.nextAbsOffset

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
	s.index.Write(uint32(s.nextAbsOffset-s.baseAbsOffset), pos)

	s.nextAbsOffset++

	return record.Offset, nil
}

func (s *segment) Read(absOff uint64) (record *api.Record, err error) {
	// 先根据相对下标读取索引文件
	// 获取记录在存储文件中的位置
	relOff := int64(absOff - s.baseAbsOffset)
	_, pos, err := s.index.Read(relOff)
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
