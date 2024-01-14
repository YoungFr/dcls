package log

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"

	api "github.com/youngfr/dcls/api/v1"
)

type Log struct {
	// 存放所有 .store 和 .index 文件的目录
	Dir    string
	Config Config

	// 保护 segments 和 activeSegment 对象
	mu sync.RWMutex

	// 从前往后按时间顺序排序的所有 segment 对象
	segments []*segment

	// 总是向其中写入的、最新的 segment 对象
	// 它总是 segments 的最后一个元素
	// 当它写满时，我们新建一个 segment 并添加在 segments 的最后边
	activeSegment *segment
}

func NewLog(dir string, c Config) (*Log, error) {
	if c.Segment.MaxStoreBytes == 0 {
		c.Segment.MaxStoreBytes = 64 * (25 + lenSize)
	}
	if c.Segment.MaxIndexBytes == 0 {
		c.Segment.MaxIndexBytes = 64 * entrySize
	}
	l := &Log{
		Dir:      dir,
		Config:   c,
		segments: make([]*segment, 0),
	}
	return l, l.setup()
}

func (l *Log) setup() error {
	// 我们在 l.Dir 目录下存储的都是 .store 和 .index 文件
	files, err := os.ReadDir(l.Dir)
	if err != nil {
		return err
	}

	baseOffsets := make([]uint64, 0)
	for _, file := range files {
		// 我们在 segment 中创建存储文件和索引文件时
		// 使用的是 xxx.store 和 xxx.index 的格式
		// 其中 xxx 都是数字，表示的是这个文件中的第一项记录的绝对下标
		//
		// 在服务启动时，我们要根据这些下标来创建对应的 segment 对象
		// 因为每个数字都有对应的 .store 和 .index 文件
		// 所以只需要从所有 .store(或.index) 文件中提取数字
		baseOffset, _ := strconv.ParseUint(
			strings.TrimSuffix(file.Name(), ".store"), 10, 0)
		baseOffsets = append(baseOffsets, baseOffset)
	}

	// 较小的下标对应较老的记录而较大的下标对应较新的记录
	// 我们需要将所有下标排序后再按序创建 segment 以保证所有记录都是按时间先后排序的
	sort.Slice(baseOffsets, func(i, j int) bool {
		return baseOffsets[i] < baseOffsets[j]
	})
	for _, baseOffset := range baseOffsets {
		if err := l.newSegment(baseOffset); err != nil {
			return err
		}
	}

	// 服务第一次启动 => 根据配置的 InitialOffset 值创建一个新的 segment 对象
	if len(l.segments) == 0 {
		if err = l.newSegment(l.Config.Segment.InitialOffset); err != nil {
			return err
		}
	}

	return nil
}

func (l *Log) newSegment(baseOffset uint64) error {
	s, err := newSegment(l.Dir, baseOffset, l.Config)
	if err != nil {
		return err
	}
	l.segments = append(l.segments, s)
	l.activeSegment = s
	return nil
}

func (l *Log) Append(record *api.Record) (offset uint64, err error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	offset, err = l.activeSegment.Append(record)
	if err != nil {
		if err != errNotEnoughIndexSpace {
			return 0, err
		} else {
			// 索引文件空间不足
			// 在 (*segment).Append 方法中已经对这种情况提前做了判断
			// 此时需要新建一个 segment 来进行写入
			err = l.newSegment(offset + 1)
			if err != nil {
				return 0, err
			}

			offset, err = l.activeSegment.Append(record)
			// 在新建的 segment 中调用 Append 不会发生 errNotEnoughIndexSpace 错误
			// 此时 err 不为空表示写入失败了
			if err != nil {
				return 0, err
			}
		}
	} else {
		// 错误为 nil 但是存储文件的大小达到了最大值
		// 需要以 offset+1 为 baseOffset 新建一个 segment 对象
		// 这里无论新建 segment 是否成功都需要返回 offset 而不是零
		// 因为我们在旧的 segment 中已经成功添加了记录和索引
		if l.activeSegment.IsMaxed() {
			err = l.newSegment(offset + 1)
			return offset, err
		}
	}

	return offset, nil
}

func (l *Log) Read(offset uint64) (record *api.Record, err error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	var s *segment
	for _, segment := range l.segments {
		if segment.baseOffset <= offset && offset < segment.nextOffset {
			s = segment
			break
		}
	}

	if s == nil || s.nextOffset <= offset {
		return nil, fmt.Errorf("offset out of range: %d", offset)
	}

	return s.Read(offset)
}

func (l *Log) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	for _, segment := range l.segments {
		if err := segment.Close(); err != nil {
			return err
		}
	}
	return nil
}

func (l *Log) Remove() error {
	if err := l.Close(); err != nil {
		return err
	}
	return os.RemoveAll(l.Dir)
}

func (l *Log) Reset() error {
	if err := l.Remove(); err != nil {
		return err
	}
	return l.setup()
}

func (l *Log) LowestOffset() (uint64, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	return l.segments[0].baseOffset, nil
}

func (l *Log) HighestOffset() (uint64, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	offset := l.segments[len(l.segments)-1].nextOffset
	if offset == 0 {
		return 0, nil
	}
	return offset - 1, nil
}

func (l *Log) Truncate(lowest uint64) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	segments := make([]*segment, 0)
	for _, s := range l.segments {
		if s.nextOffset <= lowest+1 {
			if err := s.Remove(); err != nil {
				return err
			}
			continue
		}
		segments = append(segments, s)
	}
	l.segments = segments
	return nil
}
