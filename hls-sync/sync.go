package main

import (
	"bytes"
	"github.com/grafov/m3u8"
	log "github.com/Sirupsen/logrus"
	"os"
)

type SyncType uint8

const (
	PLAYLIST SyncType = 1 + iota
	SEGMEMT
)

type SyncMessage struct {
	_type SyncType
	playlist *m3u8.MediaPlaylist
	segment  *m3u8.MediaSegment
	seg_buffer *bytes.Buffer
}

func deleteOldSegment(filename string) {
	log.Printf("deleteOldSegment:> %v\n", filename)
	err := os.Remove(filename)
	if err != nil {
		log.Printf("Delete file %v failed! <%v>", filename, err)
	}
}


func (self *Synchronizer) syncProc(msgChan chan *SyncMessage) {
	for msg := range msgChan {
		if nil == msg {
			continue
		}
		switch msg._type {
		case PLAYLIST:
			log.Debugln("Syncing playlist:> ", msg.playlist.SeqNo, len(msg.playlist.Segments))
			//break
		case SEGMEMT:
			log.Debugln("Syncing segment:> ", msg.segment.URI, msg.seg_buffer.Len())
			//break
		}
	}
}