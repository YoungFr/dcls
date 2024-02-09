package tests

import (
	"os"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
	api "github.com/youngfr/dcls/api/v1"
	dclslog "github.com/youngfr/dcls/internal/log"
)

func TestLogOperations(t *testing.T) {
	t.Run("log operations test", func(t *testing.T) {
		// averageRecordLength = 6
		// 使用的测试数据的格式是
		// &api.Record{Offset: x, Value: []byte(strconv.Itoa(x))} (x <= 10 <= 99)
		// 使用 proto.Size 方法可知其长度为 6 个字节

		// -------------------- TestCase 1 --------------------
		dir1, err := os.MkdirTemp("", "clog1")
		require.NoError(t, err)
		clog1, err := dclslog.NewLog(dir1, dclslog.Config{
			Segment: struct {
				MaxStoreBytes uint64
				MaxIndexBytes uint64
				InitialOffset uint64
			}{
				MaxStoreBytes: 90 * (6 + 8),
				MaxIndexBytes: 90 * 12,
				InitialOffset: 10,
			},
		})
		require.NoError(t, err)
		defer clog1.Close()
		for i := uint64(10); i < uint64(100); i++ {
			off, err := clog1.Append(&api.Record{
				Value: []byte(strconv.Itoa(int(i))),
			})
			require.NoError(t, err)
			require.Equal(t, i, off)
		}
		for i := uint64(10); i < uint64(100); i++ {
			record, err := clog1.Read(i)
			require.NoError(t, err)
			require.Equal(t, i, record.Offset)
			require.Equal(t, []byte(strconv.Itoa(int(i))), record.Value)
		}
		files1, err := os.ReadDir(dir1)
		require.NoError(t, err)
		require.Equal(t, 4, len(files1))
		// -------------------- TestCase 1 --------------------

		// -------------------- TestCase 2 --------------------
		dir2, err := os.MkdirTemp("", "clog2")
		require.NoError(t, err)
		clog2, err := dclslog.NewLog(dir2, dclslog.Config{
			Segment: struct {
				MaxStoreBytes uint64
				MaxIndexBytes uint64
				InitialOffset uint64
			}{
				MaxStoreBytes: 91 * (6 + 8),
				MaxIndexBytes: 91 * 12,
				InitialOffset: 10,
			},
		})
		require.NoError(t, err)
		defer clog2.Close()
		for i := uint64(10); i < uint64(100); i++ {
			off, err := clog2.Append(&api.Record{
				Value: []byte(strconv.Itoa(int(i))),
			})
			require.NoError(t, err)
			require.Equal(t, i, off)
		}
		files2, err := os.ReadDir(dir2)
		require.NoError(t, err)
		require.Equal(t, 2, len(files2))
		// -------------------- TestCase 2 --------------------

		// -------------------- TestCase 3 --------------------
		dir3, err := os.MkdirTemp("", "clog3")
		require.NoError(t, err)
		clog3, err := dclslog.NewLog(dir3, dclslog.Config{
			Segment: struct {
				MaxStoreBytes uint64
				MaxIndexBytes uint64
				InitialOffset uint64
			}{
				MaxStoreBytes: 10 * (6 + 8),
				MaxIndexBytes: 10 * 12,
				InitialOffset: 10,
			},
		})
		require.NoError(t, err)
		for i := uint64(10); i < uint64(100); i++ {
			off, err := clog3.Append(&api.Record{
				Value: []byte(strconv.Itoa(int(i))),
			})
			require.NoError(t, err)
			require.Equal(t, i, off)
		}
		files3, err := os.ReadDir(dir3)
		require.NoError(t, err)
		require.Equal(t, 20, len(files3))
		// -------------------- TestCase 3 --------------------

		// -------------------- TestCase 4 --------------------
		dir4, err := os.MkdirTemp("", "clog4")
		require.NoError(t, err)
		clog4, err := dclslog.NewLog(dir4, dclslog.Config{
			Segment: struct {
				MaxStoreBytes uint64
				MaxIndexBytes uint64
				InitialOffset uint64
			}{
				MaxStoreBytes: 20 * (6 + 8),
				MaxIndexBytes: 10 * 12,
				InitialOffset: 10,
			},
		})
		require.NoError(t, err)
		for i := uint64(10); i < uint64(100); i++ {
			off, err := clog4.Append(&api.Record{
				Value: []byte(strconv.Itoa(int(i))),
			})
			require.NoError(t, err)
			require.Equal(t, i, off)
		}
		files4, err := os.ReadDir(dir4)
		require.NoError(t, err)
		require.Equal(t, 20, len(files4))
		// -------------------- TestCase 4 --------------------

		// -------------------- TestCase 5 --------------------
		dir5, err := os.MkdirTemp("", "clog5")
		require.NoError(t, err)
		clog5, err := dclslog.NewLog(dir5, dclslog.Config{
			Segment: struct {
				MaxStoreBytes uint64
				MaxIndexBytes uint64
				InitialOffset uint64
			}{
				MaxStoreBytes: 10 * (6 + 8),
				MaxIndexBytes: 20 * 12,
				InitialOffset: 10,
			},
		})
		require.NoError(t, err)
		for i := uint64(10); i < uint64(100); i++ {
			off, err := clog5.Append(&api.Record{
				Value: []byte(strconv.Itoa(int(i))),
			})
			require.NoError(t, err)
			require.Equal(t, i, off)
		}
		files5, err := os.ReadDir(dir5)
		require.NoError(t, err)
		require.Equal(t, 20, len(files5))
		// -------------------- TestCase 5 --------------------
	})
}
