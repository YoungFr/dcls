package log

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"

	api "github.com/youngfr/dcls/api/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Log struct {
	// 存放所有 .store 和 .index 文件的目录
	Dir    string
	Config Config

	// 保护 segments 和 activeSegment 对象
	mu sync.RWMutex

	// 从前往后按时间顺序排序的所有 segment 对象
	segments []*segment

	// 最新的 segment 对象，我们总是向其中写入数据
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

	baseAbsOffsets := make([]uint64, 0)
	for _, file := range files {
		// 我们在 segment 中创建存储文件和索引文件时
		// 使用的是 xxx.store 和 xxx.index 的格式
		// 其中 xxx 表示的是该 segment 存储的第一项记录的绝对下标
		//
		// 在服务启动时，我们要根据这些下标来创建对应的 segment 对象
		// 因为每个绝对下标都有对应的 .store 和 .index 文件
		// 所以只需要从所有 .store(或.index) 文件中提取数字
		// 我们这里从所有 .store 文件中提取
		if strings.Contains(file.Name(), ".store") {
			baseAbsOffset, _ := strconv.ParseUint(
				strings.TrimSuffix(file.Name(), ".store"), 10, 0)
			baseAbsOffsets = append(baseAbsOffsets, baseAbsOffset)
		}
	}

	// 较小的下标对应较老的记录而较大的下标对应较新的记录
	// 我们需要将所有下标排序后再按序创建 segment 以保证所有记录都是按时间先后排序的
	sort.Slice(baseAbsOffsets, func(i, j int) bool {
		return baseAbsOffsets[i] < baseAbsOffsets[j]
	})
	for _, baseAbsOffset := range baseAbsOffsets {
		if err := l.newSegment(baseAbsOffset); err != nil {
			return err
		}
	}

	// 还没有 segment 则根据配置的 InitialOffset 值创建一个新的 segment 对象
	if len(l.segments) == 0 {
		if err = l.newSegment(l.Config.Segment.InitialOffset); err != nil {
			return err
		}
	}

	return nil
}

func (l *Log) newSegment(baseAbsOffset uint64) error {
	s, err := newSegment(l.Dir, baseAbsOffset, l.Config)
	if err != nil {
		return err
	}
	l.segments = append(l.segments, s)
	l.activeSegment = s
	return nil
}

func (l *Log) Append(record *api.Record) (absOff uint64, err error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	absOff, err = l.activeSegment.Append(record)
	if err != nil {
		if err != errNotEnoughIndexSpace {
			return 0, err
		} else {
			// 索引文件空间不足
			// 在 (*segment).Append 方法中已经对这种情况提前做了判断
			// 此时需要新建一个 segment 来进行写入
			err = l.newSegment(absOff + 1)
			if err != nil {
				return 0, err
			}

			absOff, err = l.activeSegment.Append(record)
			// 在新建的 segment 中调用 Append 不会发生 errNotEnoughIndexSpace 错误
			// 此时 err 不为空表示写入失败了
			if err != nil {
				return 0, err
			}
		}
	} else {
		// 错误为 nil 但是 segment 的 store(或index) 达到了最大值
		// 需要以 offset+1 为 baseAbsOffset 新建一个 segment 对象
		// 以保证下次写入时能写入正确的 segment 中
		//
		// 注意：在这里无论新建 segment 是否成功都需要返回 absOff 而不是零
		// 因为我们在旧的 segment 中已经成功添加了记录和索引
		if l.activeSegment.IsMaxed() {
			err = l.newSegment(absOff + 1)
			return absOff, err
		}
	}

	return absOff, nil
}

func (l *Log) Read(absOff uint64) (record *api.Record, err error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	var s *segment
	for _, segment := range l.segments {
		if segment.baseAbsOffset <= absOff && absOff < segment.nextAbsOffset {
			s = segment
			break
		}
	}

	if s == nil || s.nextAbsOffset <= absOff {
		return nil, status.Error(
			codes.InvalidArgument,
			fmt.Sprintf("offset out of range: %d", absOff),
		)
	}

	return s.Read(absOff)
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

func (l *Log) Reset() error {
	if err := l.Close(); err != nil {
		return err
	}

	// 删除日志存储目录中下所有文件
	files, err := os.ReadDir(l.Dir)
	if err != nil {
		return err
	}
	for _, file := range files {
		if err := os.Remove(
			l.Dir + string(os.PathSeparator) + file.Name(),
		); err != nil {
			return err
		}
	}

	l.segments = make([]*segment, 0)
	l.activeSegment = nil

	return l.setup()
}
