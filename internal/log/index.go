package log

import (
	"errors"
	"math"
	"os"

	"github.com/tysonmote/gommap"
)

const (
	// 每条记录都有一个 offset 域表示它在所有记录列表中的绝对下标
	// 在每个 index 结构体中都会保存这个下标，但是是以相对下标的形式保存
	// 即每个 index 中都是 0, 1, 2, ... 的形式，就像下边这样：
	//
	// records:   0   1   2     3   4   5     6   7   8
	//          +---+---+---+ +---+---+---+ +---+---+---+
	// indexs:  | 0 | 1 | 2 | | 0 | 1 | 2 | | 0 | 1 | 2 |
	//          +---+---+---+ +---+---+---+ +---+---+---+
	//             index1        index2        index3
	//
	// 使用 uint32 类型来存储相对下标以节省空间，占用 4 个字节
	relOffSize = 4 // sizeof(uint32)

	// 该记录从文件的第几个字节开始存储
	// 由 (*store).Append 方法返回的 pos 占用 8 个字节
	posSize = 8 // sizeof(uint64)

	entrySize = relOffSize + posSize
)

type index struct {
	file *os.File
	mmap gommap.MMap
	size uint64
}

func newIndex(f *os.File, c Config) (*index, error) {
	idx := &index{file: f}

	// 获取文件的初始大小
	finfo, err := os.Stat(f.Name())
	if err != nil {
		return nil, err
	}
	idx.size = uint64(finfo.Size())

	// 因为一旦内存映射完成，其大小就不能再更改
	// 所以在映射前需要先将文件的大小截断为 MaxIndexBytes 个字节
	if err = os.Truncate(
		f.Name(),
		int64(c.Segment.MaxIndexBytes)); err != nil {
		return nil, err
	}

	// 进行内存映射
	if idx.mmap, err = gommap.Map(
		f.Fd(),
		gommap.PROT_READ|gommap.PROT_WRITE,
		// 共享文件映射 => 内存映射 I/O
		gommap.MAP_SHARED); err != nil {
		return nil, err
	}

	return idx, nil
}

var errEmptyIndexFile = errors.New("index file is empty")
var errInvalidRelativeOffset = errors.New("invalid relative offset")

// 根据相对下标查询对应的记录是从文件的第几个字节开始存储的
//
// 将相对下标既作为输入又作为输出的原因是
// 当输入为 -1 时返回的是当前最后一个索引项对应的相对下标
// 也就是说可以用 Read(-1) 来获取当前索引项的数目
//
// 因为要接收负数作为输入，所以 refOffInput 的类型为
// 能容纳所有 uint32 数字的有符号整型即 int64 类型
func (i *index) Read(relOffInput int64) (relOffOutput uint32, pos uint64, err error) {
	if i.size == 0 {
		return 0, 0, errEmptyIndexFile
	}
	// 当前最大的相对下标
	// 参数 relOffInput 的值必须在 [-1, currMaxRelOff] 范围内
	currMaxRelOff := uint32(i.size/entrySize - 1)
	if relOffInput < -1 {
		return 0, 0, errInvalidRelativeOffset
	}
	if relOffInput >= 0 && (relOffInput > math.MaxUint32 || uint32(relOffInput) > currMaxRelOff) {
		return 0, 0, errInvalidRelativeOffset
	}
	if relOffInput == -1 {
		relOffOutput = currMaxRelOff
	} else {
		relOffOutput = uint32(relOffInput)
	}
	entryBeginIndex := relOffOutput * entrySize
	relOffOutput = order.Uint32(i.mmap[entryBeginIndex : entryBeginIndex+relOffSize])
	pos = order.Uint64(i.mmap[entryBeginIndex+relOffSize : entryBeginIndex+entrySize])
	return relOffOutput, pos, nil
}

// 判断 index 是否还有空间存储一个新的索引项
func (i *index) HasSpace() bool {
	return i.size+entrySize <= uint64(len(i.mmap))
}

var errNotEnoughIndexSpace = errors.New("index space is not enough to put new entry")

func (i *index) Write(relOff uint32, pos uint64) error {
	if !i.HasSpace() {
		return errNotEnoughIndexSpace
	}
	order.PutUint32(i.mmap[i.size:i.size+relOffSize], relOff)
	order.PutUint64(i.mmap[i.size+relOffSize:i.size+entrySize], pos)
	i.size += entrySize
	return nil
}

func (i *index) Close() error {
	if err := i.mmap.Sync(gommap.MS_SYNC); err != nil {
		return err
	}
	if err := i.file.Sync(); err != nil {
		return err
	}
	if err := i.file.Truncate(int64(i.size)); err != nil {
		return err
	}
	return nil
}

func (i *index) Name() string {
	return i.file.Name()
}
