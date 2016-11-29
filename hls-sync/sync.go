package main

import (
	"bytes"
	"github.com/archsh/m3u8"
	log "github.com/Sirupsen/logrus"
	"os"
	"path/filepath"
	"io"
	"github.com/golang/groupcache/lru"
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


func (self *Synchronizer) syncProc(msgChan chan *SyncMessage) {
	cache := lru.New(self.option.Max_Segments)
	if self.option.Sync.Remove_Old {
		cache.OnEvicted = func (k lru.Key, v interface{}){
			fname := v.(string)
			err := os.Remove(fname)
			if err != nil {
				log.Errorf("Delete file %v failed! <%v>", fname, err)
			}
		}
	}

	for msg := range msgChan {
		if nil == msg {
			continue
		}
		switch msg._type {
		case PLAYLIST:
			log.Debugln("Syncing playlist:> ", msg.playlist.SeqNo, len(msg.playlist.Segments))
			filename := filepath.Join(self.option.Sync.Output, self.option.Sync.Index_Name)
			out, err := os.Create(filename)
			if err != nil {
				log.Errorf("syncProc:> %v \n", err)
				continue
			}
			buf := msg.playlist.Encode()
			io.Copy(out, buf)
			out.Close()
		case SEGMEMT:
			log.Debugln("Syncing segment:> ", msg.segment.URI, msg.seg_buffer.Len())
			filename := filepath.Join(self.option.Sync.Output, msg.segment.URI)
			out, err := os.Create(filename)
			if err != nil {
				log.Errorf("syncProc:> %v \n", err)
				continue
			}
			n, e := msg.seg_buffer.WriteTo(out)
			if e != nil {
				log.Errorln("Write segment data failed:> ", e)
			}else{
				log.Debugln("Write segment data bytes:> ", n)
			}
			cache.Add(msg.segment.URI, filename)
			out.Close()
		}
	}
}