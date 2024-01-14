package log

type Config struct {
	Segment struct {
		MaxStoreBytes uint64 // N * (averageRecordLength + lenSize)
		MaxIndexBytes uint64 // N * entrySize
		InitialOffset uint64
	}
}
