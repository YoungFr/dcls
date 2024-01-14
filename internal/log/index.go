package log

import (
	"os"

	"github.com/tysonmote/gommap"
)

var (
	offSize   uint64 = 4
	posSize   uint64 = 8
	entrySize        = offSize + posSize
)

type index struct {
	file *os.File
	mmap gommap.MMap
	size uint64
}

func newIndex(f *os.File, c Config) (*index, error) {
	idx := &index{file: f}
	finfo, err := os.Stat(f.Name())
	if err != nil {
		return nil, err
	}
	idx.size = uint64(finfo.Size())
	if err = os.Truncate(
		f.Name(),
		int64(c.Segment.MaxIndexBytes)); err != nil {
		return nil, err
	}
	if idx.mmap, err = gommap.Map(
		f.Fd(),
		gommap.PROT_READ|gommap.PROT_WRITE,
		gommap.MAP_SHARED); err != nil {
		return nil, err
	}
	return idx, nil
}
