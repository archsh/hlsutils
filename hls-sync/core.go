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
)

type Synchronizer struct {
	option *Option
	client *http.Client
}

type SegmentMessage struct {
	segment *m3u8.MediaSegment
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
		self.playlistProc(segmentChan, syncChan, recordChan)
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

func (self *Synchronizer) playlistProc(segmentChan chan *SegmentMessage, syncChan chan *SyncMessage, recordChan chan *RecordMessage) {
	//startTime := time.Now()
	//var recDuration time.Duration = 0
	//var firstList = true
	cache := lru.New(self.option.Max_Segments)
	if self.option.Sync.Remove_Old {
		cache.OnEvicted = func(key lru.Key, value interface{}) {
			fname, res := value.(string)
			if res {
				deleteOldSegment(fname)
			}
		}
	}
	retry := 0
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
		if listType == m3u8.MEDIA {
			mpl := playlist.(*m3u8.MediaPlaylist)
			//log.Infoln("Get playlist:> ", mpl.SeqNo, mpl.TargetDuration, len(mpl.Segments))
			//log.Debugln("Cache length:> ", cache.Len())
			for _, v := range mpl.Segments {
				if v != nil {
					//var msURI string
					//var msFilename string
					//if strings.HasPrefix(v.URI, "http") || strings.HasPrefix(v.URI, "https") {
					//	msURI, err = url.QueryUnescape(v.URI)
					//	if err != nil {
					//		log.Println(err)
					//		// log.Fatal(err)
					//	}
					//	//msFilename = path.Join(path.Dir(playlistFilename), path.Base(msURI))
					//} else {
					//	msUrl, err := resp.Request.URL.Parse(v.URI)
					//	if err != nil {
					//		log.Print(err)
					//		continue
					//	}
					//	msURI, err = url.QueryUnescape(msUrl.String())
					//	if err != nil {
					//		log.Println(err)
					//		// log.Fatal(err)
					//	}
					//	//msFilename = path.Join(outDir, msUrl.Path)
					//}
					_, hit := cache.Get(v.URI)
					if !hit {
						cache.Add(v.URI, nil)
						log.Infoln("New segment:> ", mpl.SeqNo, v.URI, v.Duration, v.SeqId, v.ProgramDateTime)
						if self.option.Sync.Enabled || self.option.Record.Enabled {
							// Only get segments when sync or record enabled.
							msg := &SegmentMessage{}
							msg.segment = v
							msg.response = resp
							segmentChan <- msg
						}
					}
				}
			}
			if self.option.Sync.Enabled {
				msg := &SyncMessage{}
				msg._type = PLAYLIST
				msg.playlist = mpl
				syncChan <- msg
			}
			//if !firstList {
			//	cache.RemoveOldest()
			//}
			//firstList = false
			if mpl.Closed {
				close(syncChan)
				close(recordChan)
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
		log.Debugln("Getting segment:> ", msg.segment.URI)
		var msURI string
		if strings.HasPrefix(msg.segment.URI, "http://") || strings.HasPrefix(msg.segment.URI, "https://") {
			msURI, _ = url.QueryUnescape(msg.segment.URI)
		} else {
			msUrl, _ := msg.response.Request.URL.Parse(msg.segment.URI)
			msURI, _ = url.QueryUnescape(msUrl.String())
		}
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
			if self.option.Sync.Enabled {
				le_msg := &SyncMessage{}
				le_msg._type = SEGMEMT
				le_msg.segment = msg.segment
				le_msg.seg_buffer = buffer
				syncChan <- le_msg
			}
			if self.option.Record.Enabled {
				le_msg := &RecordMessage{}
				le_msg.segment = msg.segment
				le_msg.seg_buffer = buffer
				recordChan <- le_msg
			}
		}
	}
}


func (self *Synchronizer) doRequest(req *http.Request) (*http.Response, error) {
	req.Header.Set("User-Agent", self.option.User_Agent)
	resp, err := self.client.Do(req)
	return resp, err
}

