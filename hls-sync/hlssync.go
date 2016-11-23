package main

import (
	"bytes"
	"fmt"
	"github.com/golang/groupcache/lru"
	"github.com/kz26/m3u8"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"time"
	log "github.com/Sirupsen/logrus"
)


type HLSSynchronizer struct {
	sync_option *SyncOption
}

func NewHLSSynchronizer(lko *SyncOption) *HLSSynchronizer{
	hlssynchronizer := new(HLSSynchronizer)
	hlssynchronizer.sync_option = lko
	return hlssynchronizer
}

func (self *HLSSynchronizer) Run() {
	log.Infoln("HLSSynchronizer.Run > Start sync", self.sync_option.Source)
}


var USER_AGENT string

var client = &http.Client{}

func doRequest(c *http.Client, req *http.Request) (*http.Response, error) {
	req.Header.Set("User-Agent", USER_AGENT)
	resp, err := c.Do(req)
	return resp, err
}

type Download struct {
	URI           string
	totalDuration time.Duration
	Filename      string
}

type Status struct {
	sourceUri  string
	statusCode uint
	respBody   string
	errMsg     string
	timeStamp  time.Time
}

func getsaveSegment(url string, filename string) (string, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Println(err)
		// log.Fatal(err)
	}

	err = os.MkdirAll(path.Dir(filename), 0777)
	if err != nil {
		log.Println(err)
		// log.Fatal(err)
	}

	out, err := os.Create(filename)
	if err != nil {
		log.Println(err)
		// log.Fatal(err)
	}
	defer out.Close()
	resp, err := doRequest(client, req)
	if err != nil {
		log.Println(err)
		// log.Fatal(err)
	}
	if resp.StatusCode != 200 {
		log.Printf("Received HTTP %v for %v \n", resp.StatusCode, url)
	}
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		log.Println(err)
		// log.Fatal(err)
	}
	resp.Body.Close()
	log.Printf("Downloaded %v into %v\n", url, filename)
	return filename, err
}

func downloadSegment(dlc chan *Download, recTime time.Duration) {
	for v := range dlc {
		_, err := getsaveSegment(v.URI, v.Filename)
		if err != nil {
			log.Println(err)
			// log.Fatal(err)
		}
		if recTime != 0 {
			log.Printf("Recorded %v of %v\n", v.totalDuration, recTime)
		} else {
			log.Printf("Recorded %v\n", v.totalDuration)
		}
	}
}

func deleteOldSegment(filename string) {
	log.Printf("deleteOldSegment:> %v\n", filename)
	err := os.Remove(filename)
	if err != nil {
		log.Printf("Delete file %v failed! <%v>", filename, err)
	}
}

func getPlaylist(urlStr string, outDir string, recTime time.Duration, deleteOld bool, useLocalTime bool, retry int, dlc chan *Download) {
	startTime := time.Now()
	var recDuration time.Duration = 0
	var firstList = true
	cache := lru.New(1024)
	if deleteOld {
		cache.OnEvicted = func(key lru.Key, value interface{}) {
			fname, res := value.(string)
			if res {
				deleteOldSegment(fname)
			}
		}
	}

	outPath, err := os.Open(outDir)
	if err != nil {
		log.Println(err)
		// log.Fatal(err)
		return
	}
	defer outPath.Close()
	fstat, err := outPath.Stat()
	if err != nil {
		log.Println(err)
		// log.Fatal(err)
		return
	}
	if fstat.IsDir() != true {
		log.Println("Output is not a directory!")
		// log.Fatal("Output is not a directory!")
		return
	}

	//	playlistUrl, err := url.Parse(urlStr)
	//	if err != nil {
	//		log.Fatal(err)
	//	}

	for {
		req, err := http.NewRequest("GET", urlStr, nil)
		if err != nil {
			log.Println(err)
			// log.Fatal(err)
			return
		}
		resp, err := doRequest(client, req)
		if err != nil {
			log.Println(err)
			time.Sleep(time.Duration(3) * time.Second)
			continue
		}
		respBody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Println(err)
			// log.Fatal(err)
			time.Sleep(time.Duration(3) * time.Second)
			continue
		}
		buffer := bytes.NewBuffer(respBody)
		playlistFilename := path.Join(outDir, resp.Request.URL.Path)
		err = os.MkdirAll(path.Dir(playlistFilename), 0777)
		if err != nil {
			log.Println(err)
			// log.Fatal(err)
			return
		}
		playlist, listType, err := m3u8.Decode(*buffer, true)
		//		m3u8.DecodeFrom(resp.Body, true)
		if err != nil {
			log.Println(err)
			// log.Fatal(err)
			time.Sleep(time.Duration(3) * time.Second)
			if retry < 0 {
				continue
			} else if retry > 0 {
				retry--
				continue
			} else {
				return
			}
		}
		resp.Body.Close()
		if listType == m3u8.MEDIA {
			mpl := playlist.(*m3u8.MediaPlaylist)
			for _, v := range mpl.Segments {
				if v != nil {
					var msURI string
					var msFilename string
					if strings.HasPrefix(v.URI, "http") {
						msURI, err = url.QueryUnescape(v.URI)
						if err != nil {
							log.Println(err)
							// log.Fatal(err)
						}
						msFilename = path.Join(path.Dir(playlistFilename), path.Base(msURI))
					} else {
						msUrl, err := resp.Request.URL.Parse(v.URI)
						if err != nil {
							log.Print(err)
							continue
						}
						msURI, err = url.QueryUnescape(msUrl.String())
						if err != nil {
							log.Println(err)
							// log.Fatal(err)
						}
						msFilename = path.Join(outDir, msUrl.Path)
					}
					_, hit := cache.Get(msURI)
					if !hit {
						cache.Add(msURI, msFilename)
						if useLocalTime {
							recDuration = time.Now().Sub(startTime)
						} else {
							recDuration += time.Duration(int64(v.Duration * 1000000000))
						}
						dlc <- &Download{msURI, recDuration, msFilename}
					}
					if recTime != 0 && recDuration != 0 && recDuration >= recTime {
						close(dlc)
						return
					}
				}
			}
			out, err := os.Create(playlistFilename)
			if err != nil {
				log.Println(err)
				// log.Fatal(err)
			}
			io.Copy(out, buffer)
			out.Close()
			if !firstList {
				cache.RemoveOldest()
			}
			firstList = false
			if mpl.Closed {
				close(dlc)
				return
			} else {
				time.Sleep(time.Duration(int64((mpl.TargetDuration/2) * 1000000000)))
			}
		} else {
			log.Println("Not a valid media playlist")
			if retry < 0 {
				continue
			} else if retry > 0 {
				retry--
				continue
			} else {
				return
			}
		}
	}
}

func update_status(stc chan *Status, stm map[string]*Status) {

	for s := range stc {
		fmt.Println(s.sourceUri)
		stm[s.sourceUri] = s
	}

}
