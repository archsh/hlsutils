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

func (self *Synchronizer) recordProc(msgChan chan *RecordMessage) {
	for msg := range msgChan {
		if nil == msg {
			continue
		}
		log.Debugln("Recording segment:> ", msg.segment, msg.seg_buffer.Len())
	}
}