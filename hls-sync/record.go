package main

import (
	"bytes"
	"github.com/archsh/m3u8"
	log "github.com/Sirupsen/logrus"
	"strings"
	"github.com/archsh/timefmt"
	"time"
	"regexp"
	"fmt"
	"os"
	"io"
	"path/filepath"
)

type RecordMessage struct {
	_target_duration float64
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
	index_by := IDXT_HOUR
	switch strings.ToLower(self.option.Record.Reindex_By) {
	case "hour":
		index_by = IDXT_HOUR
	case "minute":
		index_by = IDXT_MINUTE
	default:
		index_by = IDXT_HOUR
	}
	log.Debugln("Index By:", index_by)
	index := 0
	index_playlist, e := m3u8.NewMediaPlaylist(2048, 2048)
	if nil != e {
		log.Errorln("Create playlist failed:>", e)
	}
	last_seg_timestamp := time.Time{}
	for msg := range msgChan {
		if nil == msg {
			continue
		}
		segtime := msg.segment.ProgramDateTime
		if index_by == IDXT_MINUTE {
			if segtime.Year() != last_seg_timestamp.Year() ||
				segtime.Month() != last_seg_timestamp.Month() ||
				segtime.Day() != last_seg_timestamp.Day() ||
				segtime.Hour() != last_seg_timestamp.Hour() ||
				segtime.Minute() != last_seg_timestamp.Minute() {
				if self.option.Target_Duration < 1 {
					index = segtime.Second()/int(msg._target_duration)
				}else{
					index = segtime.Second()/self.option.Target_Duration
				}
				index_playlist.Close()
				self.saveIndexPlaylist(index_playlist)
				index_playlist, e = m3u8.NewMediaPlaylist(2048, 2048)
				if nil != e {
					log.Errorln("Create playlist failed:>", e)
					continue
				}
				if self.option.Target_Duration < 1 {
					index_playlist.TargetDuration = msg._target_duration
				}else{
					index_playlist.TargetDuration = float64(self.option.Target_Duration)
				}
			}
		}else{
			if segtime.Year() != last_seg_timestamp.Year() ||
				segtime.Month() != last_seg_timestamp.Month() ||
				segtime.Day() != last_seg_timestamp.Day() ||
				segtime.Hour() != last_seg_timestamp.Hour() {
				if self.option.Target_Duration < 1 {
					index = (segtime.Minute()*60+segtime.Second())/int(msg._target_duration)
				}else{
					index = (segtime.Minute()*60+segtime.Second())/self.option.Target_Duration
				}
				index_playlist.Close()
				self.saveIndexPlaylist(index_playlist)
				index_playlist, e = m3u8.NewMediaPlaylist(2048, 2048)
				if nil != e {
					log.Errorln("Create playlist failed:>", e)
					continue
				}
				if self.option.Target_Duration < 1 {
					index_playlist.TargetDuration = msg._target_duration
				}else{
					index_playlist.TargetDuration = float64(self.option.Target_Duration)
				}
			}
		}
		log.Debugln("Recording segment:> ", msg.segment, msg.seg_buffer.Len())
		fname, e := self.generateFilename(self.option.Record.Output, self.option.Record.Segment_Rewrite, msg.segment.ProgramDateTime, index)
		log.Debugf("New filename:> %s <%s> \n", fname, e)
		e = os.MkdirAll(filepath.Dir(fname), 0777)
		if e != nil {
			log.Errorf("Create directory '%s' failed: %v \n", filepath.Dir(fname), e)
			continue
		}
		out, err := os.Create(fname)
		if err != nil {
			log.Errorf("Create segment file failed:> %v \n", err)
			return
		}
		n, e := msg.seg_buffer.WriteTo(out)
		if nil != e {
			log.Errorf("Write to segment file failed:> %v \n", err)
			out.Close()
			continue
		}else{
			log.Debugln("Write to segment file bytes: ", n)
		}
		out.Close()
		last_seg_timestamp = msg.segment.ProgramDateTime
		index++
		//msg.segment.URI = filepath.Base(fname)
		//index_playlist.AppendSegment(msg.segment)
		seg := m3u8.MediaSegment{
			URI: filepath.Base(fname),
			Duration: msg.segment.Duration,
			ProgramDateTime: msg.segment.ProgramDateTime,
			Title: msg.segment.URI,
		}
		index_playlist.AppendSegment(&seg)
		self.saveIndexPlaylist(index_playlist)
	}
}

func (self *Synchronizer) saveIndexPlaylist(playlist *m3u8.MediaPlaylist) {
	if nil == playlist || nil == playlist.Segments[0] {
		log.Errorln("Empty playlist !")
		return
	}
	fname, e := self.generateFilename(self.option.Record.Output, self.option.Record.Reindex_Format, playlist.Segments[0].ProgramDateTime, 0)
	log.Debugf("Re-index into file:> %s <%s> \n", fname, e)
	e = os.MkdirAll(filepath.Dir(fname), 0777)
	if e != nil {
		log.Errorf("saveIndexPlaylist:> Create directory '%s' failed: %v \n", filepath.Dir(fname), e)
		return
	}
	out, err := os.Create(fname)
	defer out.Close()
	if err != nil {
		log.Errorf("saveIndexPlaylist:> %v \n", err)
		return
	}
	buf := playlist.Encode()
	io.Copy(out, buf)
}

func (self *Synchronizer) generateFilename(output string, format string, tm time.Time, idx int) (string, error) {
	s, e := timefmt.Strftime(tm, format)
	if e != nil {
		return "", nil
	}
	re, e := regexp.Compile("(#)(:?)(\\d{0,2})")
	if e != nil {
		return "", nil
	}
	if re.MatchString(s){
		s = re.ReplaceAllString(s, "%${3}d")
		s = fmt.Sprintf(s, idx+1)
	}
	return s, nil
}