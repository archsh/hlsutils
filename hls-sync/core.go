package main

import (
	"bytes"
	//"fmt"
	"errors"
	"github.com/golang/groupcache/lru"
	"github.com/grafov/m3u8"
	//"io"
	"io/ioutil"
	"net/http"
	//"net/url"
	//"os"
	//"path"
	//"strings"
	"time"
	log "github.com/Sirupsen/logrus"
	"sync"
	//"path/filepath"
	"strings"
	"net/url"
	//"fmt"
	"github.com/archsh/timefmt"
)

type Synchronizer struct {
	option *Option
	client *http.Client
}

type SegmentMessage struct {
	_type SyncType
	_hit bool
	_target_duration float64
	playlist *m3u8.MediaPlaylist
	segment  *m3u8.MediaSegment
	response *http.Response
}

func NewSynchronizer(option *Option) (*Synchronizer, error) {
	if len(option.Source.Urls) < 1 {
		return nil, errors.New("!!! At least one source URL is required!")
	}
	synchronizer := new(Synchronizer)
	synchronizer.option = option
	synchronizer.client = &http.Client{}
	if synchronizer.option.Retries < 1 {
		synchronizer.option.Retries = 1
	}
	return synchronizer, nil
}

func (self *Synchronizer) Run() {
	log.Infoln("Synchronizer.Run > Start hls-sync ...")
	syncChan := make(chan *SyncMessage, 20)
	recordChan := make(chan *RecordMessage, 20)
	segmentChan := make(chan *SegmentMessage, 20)
	var wg sync.WaitGroup
	wg.Add(4)
	go func(){
		self.playlistProc(segmentChan)
		wg.Done()
	}()
	go func(){
		self.segmentProc(segmentChan, syncChan, recordChan)
		wg.Done()
	}()
	go func(){
		self.syncProc(syncChan)
		wg.Done()
	}()
	go func(){
		self.recordProc(recordChan)
		wg.Done()
	}()
	wg.Wait()
}

func (self *Synchronizer) playlistProc(segmentChan chan *SegmentMessage) {
	cache := lru.New(self.option.Max_Segments)
	retry := 0
	timezone_shift := time.Minute * time.Duration(self.option.Timezone_shift)
	timestamp_type := TST_LOCAL
	switch strings.ToLower(self.option.Timestamp_type) {
	case "local":
		timestamp_type = TST_LOCAL
	case "segment":
		timestamp_type = TST_SEGMENT
	default:
		timestamp_type = TST_PROGRAM
	}
	log.Debugln("Timestamp Type:", timestamp_type)
	log.Debugln("Timezone Shift:", timezone_shift)
	for {
		urlStr := self.option.Source.Urls[0]
		req, err := http.NewRequest("GET", urlStr, nil)
		if err != nil {
			log.Errorln("Create Request failed:>", err)
			return
		}
		resp, err := self.doRequest(req)
		if err != nil {
			log.Errorln("doRequest failed:> ", err)
			time.Sleep(time.Duration(1) * time.Second)
			continue
		}
		respBody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Errorln("Read Response body failed:> ", err)
			time.Sleep(time.Duration(1) * time.Second)
			continue
		}
		buffer := bytes.NewBuffer(respBody)
		playlist, listType, err := m3u8.Decode(*buffer, true)
		if err != nil {
			log.Errorln("Decode playlist failed:> ", err)
			time.Sleep(time.Duration(1) * time.Second)
			retry ++
			continue
		}
		resp.Body.Close()
		mpl_updated := false
		lastTimestamp := time.Now()
		if listType == m3u8.MEDIA {
			mpl := playlist.(*m3u8.MediaPlaylist)
			//log.Debugln("Get playlist , segments = ", len(mpl.Segments))
			for i, v := range mpl.Segments {
				if v != nil {
					//log.Debugln("Segment:> ", v.URI, v.ProgramDateTime)
					t, hit := cache.Get(v.URI)
					if !hit {
						if timestamp_type == TST_SEGMENT {
							v.ProgramDateTime, _ = timefmt.Strptime(v.URI, self.option.Timestamp_Format)
						}
						if timestamp_type == TST_LOCAL || v.ProgramDateTime.Year() < 2016 || v.ProgramDateTime.Month() == 0 || v.ProgramDateTime.Day() == 0 {
							v.ProgramDateTime = lastTimestamp
							lastTimestamp = lastTimestamp.Add(time.Duration(v.Duration)*time.Second)
						}else{
							v.ProgramDateTime = v.ProgramDateTime.Add(timezone_shift)
						}
						cache.Add(v.URI, v.ProgramDateTime)
						log.Infoln("New segment:> ", i, "=>", mpl.SeqNo, v.URI, v.Duration, v.SeqId, v.ProgramDateTime)
						if self.option.Sync.Enabled || self.option.Record.Enabled {
							// Only get segments when sync or record enabled.
							msg := &SegmentMessage{}
							msg._type = SEGMEMT
							msg._hit = false
							msg._target_duration = mpl.TargetDuration
							msg.segment = v
							msg.response = resp
							segmentChan <- msg
						}
						mpl_updated = true
					}else{
						v.ProgramDateTime = t.(time.Time)
						if self.option.Sync.Enabled || self.option.Record.Enabled {
							// Only get segments when sync or record enabled.
							msg := &SegmentMessage{}
							msg._type = SEGMEMT
							msg._hit = true
							msg._target_duration = mpl.TargetDuration
							msg.segment = v
							msg.response = resp
							segmentChan <- msg
						}
					}
				}
			}
			if self.option.Sync.Enabled && mpl_updated {
				msg := &SegmentMessage{}
				msg._type = PLAYLIST
				msg._target_duration = mpl.TargetDuration
				msg.segment = nil
				msg.response = resp
				msg.playlist = mpl
				segmentChan <- msg
			}
			if mpl.Closed {
				close(segmentChan)
				return
			} else {
				time.Sleep(time.Duration(int64((mpl.TargetDuration/2) * 1000000000)))
			}
		} else {
			log.Errorln("> Not a valid media playlist")
			retry ++
		}
	}
}


func (self *Synchronizer) segmentProc(segmentChan chan *SegmentMessage, syncChan chan *SyncMessage, recordChan chan *RecordMessage) {
	for msg := range segmentChan {
		if nil == msg {
			continue
		}
		if msg._type == PLAYLIST {
			le_msg := &SyncMessage{}
			le_msg._type = msg._type
			le_msg.playlist = msg.playlist
			le_msg.segment = nil
			le_msg.seg_buffer = nil
			syncChan <- le_msg
		}else{
			var msURI string
			var msFilename string
			if strings.HasPrefix(msg.segment.URI, "http://") || strings.HasPrefix(msg.segment.URI, "https://") {
				msURI, _ = url.QueryUnescape(msg.segment.URI)
				msFilename,_ = timefmt.Strftime(msg.segment.ProgramDateTime, "%Y%m%d-%H%M%S.ts")
			} else {
				msUrl, _ := msg.response.Request.URL.Parse(msg.segment.URI)
				msURI, _ = url.QueryUnescape(msUrl.String())
				msFilename = msg.segment.URI
				//msFilename,_ = timefmt.Strftime(msg.segment.ProgramDateTime, "%Y%m%d-%H%M%S.ts")
			}
			msg.segment.URI = msFilename
			if msg._hit {
				continue
			}
			log.Debugln("Getting new segment:> ", msg.segment.URI)
			for i:=0; i< self.option.Retries; i++ {
				req, err := http.NewRequest("GET", msURI, nil)
				if err != nil {
					log.Errorf("GetSegment:1> Create new request failed: %v\n", err)
				}

				resp, err := self.doRequest(req)
				if err != nil {
					log.Errorf("GetSegment:4> do request failed: %v\n", err)
					time.Sleep(time.Duration(1) * time.Second)
					continue
				}
				if resp.StatusCode != 200 {
					log.Errorf("Received HTTP %v for %v \n", resp.StatusCode, msURI)
					time.Sleep(time.Duration(1) * time.Second)
					continue
				}
				respBody, err := ioutil.ReadAll(resp.Body)
				if err != nil {
					log.Errorln("Read Response body failed:> ", err)
					time.Sleep(time.Duration(1) * time.Second)
					continue
				}
				resp.Body.Close()
				buffer := bytes.NewBuffer(respBody)
				bufdata := buffer.Bytes()
				if self.option.Sync.Enabled {
					le_msg := &SyncMessage{}
					le_msg._type = SEGMEMT
					le_msg.segment = msg.segment
					le_msg.seg_buffer = bytes.NewBuffer(bufdata)
					syncChan <- le_msg
				}
				if self.option.Record.Enabled {
					le_msg := &RecordMessage{}
					le_msg._target_duration = msg._target_duration
					le_msg.segment = msg.segment
					le_msg.seg_buffer = bytes.NewBuffer(bufdata)
					recordChan <- le_msg
				}
			}
		}
	}
}


func (self *Synchronizer) doRequest(req *http.Request) (*http.Response, error) {
	req.Header.Set("User-Agent", self.option.User_Agent)
	resp, err := self.client.Do(req)
	return resp, err
}

