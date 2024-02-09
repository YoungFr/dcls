package log

type Config struct {
	Segment struct {
		MaxStoreBytes uint64 // N * (averageRecordLength + 8)
		MaxIndexBytes uint64 // N * 12
		InitialOffset uint64
	}
}
