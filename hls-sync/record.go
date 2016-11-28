package main

import (
	"bytes"
	"github.com/grafov/m3u8"
	log "github.com/Sirupsen/logrus"
)

type RecordMessage struct {
	segment *m3u8.MediaSegment
	seg_buffer *bytes.Buffer
}

type TimeStampType uint8
type IndexType uint8

const (
	TST_LOCAL TimeStampType = 1 + iota
	TST_PROGRAM
	TST_SEGMENT
)

const (
	IDXT_HOUR IndexType = 1 + iota
	IDXT_MINUTE
)

func (self *Synchronizer) recordProc(msgChan chan *RecordMessage) {
	timestamp_type := TST_LOCAL
	index_by := IDXT_HOUR
	switch self.option.Record.Timestamp_type {
	case "local":
		timestamp_type = TST_LOCAL
	case "segment":
		timestamp_type = TST_SEGMENT
	case "program":
		timestamp_type = TST_PROGRAM
	default:
		timestamp_type = TST_LOCAL
	}
	switch self.option.Record.Reindex_By {
	case "hour":
		index_by = IDXT_HOUR
	case "minute":
		index_by = IDXT_MINUTE
	default:
		index_by = IDXT_HOUR
	}
	log.Debugln("Timestamp Type:", timestamp_type)
	log.Debugln("Index By:", index_by)
	for msg := range msgChan {
		if nil == msg {
			continue
		}
		log.Debugln("Recording segment:> ", msg.segment, msg.seg_buffer.Len())
	}
}