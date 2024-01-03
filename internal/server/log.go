package server

import (
	"fmt"
	"sync"
)

type Record struct {
	Value  []byte `json:"value"`
	Offset uint64 `json:"offset"`
}

type Log struct {
	mu      sync.Mutex
	records []Record
}

func NewLog() *Log {
	return &Log{
		records: make([]Record, 0),
	}
}

func (l *Log) Append(r Record) (uint64, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	r.Offset = uint64(len(l.records))
	l.records = append(l.records, r)
	return r.Offset, nil
}

var errOffsetNotFound = fmt.Errorf("offset not found")

func (l *Log) Read(offset uint64) (Record, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if offset >= uint64(len(l.records)) {
		return Record{}, errOffsetNotFound
	}
	return l.records[offset], nil
}
