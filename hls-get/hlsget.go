package main

import (
	"bytes"
	"github.com/golang/groupcache/lru"
	"github.com/gosexy/redis"
	"github.com/kz26/m3u8"
	"io"
	"io/ioutil"
	log "github.com/Sirupsen/logrus"
	"net/http"
	"net/url"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"
	"github.com/archsh/hlsutils/helpers/logging"
)

const VERSION = "0.9.4"

var USER_AGENT string

var client = &http.Client{Timeout: time.Duration(20 * time.Second)}

var logging_config = logging.LoggingConfig{Format:logging.DEFAULT_FORMAT, Level:"DEBUG"}

func exists(path string) (b bool) {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	return false
}

func doRequest(c *http.Client, req *http.Request) (*http.Response, error) {
	req.Header.Set("User-Agent", USER_AGENT)
	resp, err := c.Do(req)
	return resp, err
}

type Download struct {
	URI           string
	totalDuration time.Duration
	Filename      string
	refer         string
	totalSegments uint
	index         uint
	retries       int
}

func getSaveSegment(url string, filename string) (string, error) {
	var out *os.File
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Printf("getsaveSegment:1> Create new request failed: %v\n", err)
		return filename, err
	}

	if "" != filename {
		err = os.MkdirAll(path.Dir(filename), 0777)
		if err != nil {
			log.Printf("getsaveSegment:2> Create path %s failed :%v\n", path.Dir(filename), err)
			return filename, err
		}

		out, err = os.Create(filename)
	} else {
		out, err = ioutil.TempFile("./", "__savedTempSegment")
	}
	if err != nil {
		log.Printf("getsaveSegment:3> Create file %s failed: %v\n", filename, err)
		return filename, err
	}
	defer func() {
		if "" != filename {
			out.Close()
		} else {
			fname := out.Name()
			out.Close()
			os.Remove(fname)
		}
	}()

	resp, err := doRequest(client, req)
	if err != nil {
		log.Printf("getsaveSegment:4> do request failed: %v\n", err)
		return filename, err
	}
	if resp.StatusCode != 200 {
		log.Printf("Received HTTP %v for %v \n", resp.StatusCode, url)
		return filename, err
	}
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		log.Printf("getsaveSegment:5> Copy response body failed: %v\n", err)
		return filename, err
	}
	resp.Body.Close()
	log.Printf("Downloaded %v into %v\n", url, filename)
	return filename, err
}

func downloadSegment(dlc chan *Download, recTime time.Duration, retries int) {
	for v := range dlc {
	RETRY:
		log.Printf("downloadSegment: %v \n", v)
		fname, err := getSaveSegment(v.URI, v.Filename)
		if err != nil {
			log.Printf("downloadSegment:> %v \n", err)
			if retries < 0 || (retries > 0 && v.retries < retries) {
				v.retries += 1
				log.Printf("downloadSegment:> Retry to download %s in %d times. \n", v.URI, v.retries)
				time.Sleep(time.Duration(3) * time.Second)
				goto RETRY
			}
		}
		if recTime != 0 {
			log.Printf("Recorded %v of %v into %s\n", v.totalDuration, recTime, fname)
		} else {
			log.Printf("Recorded %v into %s\n", v.totalDuration, fname)
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

func getPlaylist(urlStr string, outDir string, recTime time.Duration, retries int, useLocalTime bool, skipExists int, dlc chan *Download) {
	startTime := time.Now()
	var playlistFilename string
	var recDuration time.Duration = 0
	// var firstList = true
	var tried = 0
	cache := lru.New(1024)
	cache.OnEvicted = func(key lru.Key, value interface{}) {
		fname, res := value.(string)
		if res {
			deleteOldSegment(fname)
		}
	}
	log.Printf("URI: %s, output: %s \n", urlStr, outDir)
	if "" != outDir {
		outPath, err := os.Open(outDir)
		if err != nil {
			log.Fatalf("getPlaylist:1> %v \n", err)
		}
		defer outPath.Close()
		fstat, err := outPath.Stat()
		if err != nil {
			log.Fatalf("getPlaylist:2> %v \n", err)
		}
		if fstat.IsDir() != true {
			log.Fatal("getPlaylist:3> Output is not a directory!")
		}
	}

	for {
		req, err := http.NewRequest("GET", urlStr, nil)
		if err != nil {
			log.Printf("getPlaylist:4> %v \n", err)
			if retries == 0 || (retries > 0 && tried >= retries) {
				return
			} else {
				tried += 1
				continue
			}
		}
		resp, err := doRequest(client, req)
		if err != nil {
			log.Print(err)
			time.Sleep(time.Duration(3) * time.Second)
			if retries == 0 || (retries > 0 && tried >= retries) {
				return
			} else {
				tried += 1
				continue
			}
		}
		respBody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Printf("getPlaylist:5> %v \n", err)
			if retries == 0 || (retries > 0 && tried >= retries) {
				return
			} else {
				tried += 1
				continue
			}
		}
		buffer := bytes.NewBuffer(respBody)
		playlistFilename = path.Join(outDir, uri2storagepath(resp.Request.URL.Path))
		if "" != outDir {
			err = os.MkdirAll(path.Dir(playlistFilename), 0777)
			if err != nil {
				log.Fatalf("getPlaylist:6> %v \n", err)
			}
		}
		playlist, listType, err := m3u8.Decode(*buffer, true)
		//		m3u8.DecodeFrom(resp.Body, true)
		if err != nil {
			log.Printf("getPlaylist:7> %v \n", err)
			if retries == 0 || (retries > 0 && tried >= retries) {
				return
			} else {
				tried += 1
				continue
			}
		}
		resp.Body.Close()
		if listType == m3u8.MEDIA {
			mpl := playlist.(*m3u8.MediaPlaylist)
			segs := len(mpl.Segments)
			for idx, v := range mpl.Segments {
				if v != nil {
					var msURI string
					var msFilename string
					if strings.HasPrefix(v.URI, "http") {
						msURI, err = url.QueryUnescape(v.URI)
						if err != nil {
							log.Printf("getPlaylist:8> %v \n", err)
							if retries == 0 || (retries > 0 && tried >= retries) {
								return
							} else {
								tried += 1
								continue
							}
						}
						msFilename = path.Join(path.Dir(playlistFilename), uri2storagepath(path.Base(msURI)))
					} else {
						msUrl, err := resp.Request.URL.Parse(v.URI)
						if err != nil {
							log.Print(err)
							continue
						}
						msURI, err = url.QueryUnescape(msUrl.String())
						if err != nil {
							log.Printf("getPlaylist:9> %v \n", err)
							if retries == 0 || (retries > 0 && tried >= retries) {
								return
							} else {
								tried += 1
								continue
							}
						}
						msFilename = path.Join(outDir, uri2storagepath(msUrl.Path))
					}
					_, hit := cache.Get(msURI)
					if skipExists != 0 && exists(msFilename) {
						log.Printf("Segment [%s] exists!", msFilename)
					} else if !hit {
						cache.Add(msURI, msFilename)
						if useLocalTime {
							recDuration = time.Now().Sub(startTime)
						} else {
							recDuration += time.Duration(int64(v.Duration * 1000000000))
						}
						if "" == outDir {
							msFilename = ""
						}
						dlc <- &Download{msURI, recDuration, msFilename, urlStr, uint(segs), uint(idx + 1), 0}
					}
					if recTime != 0 && recDuration != 0 && recDuration >= recTime {
						close(dlc)
						return
					}
				}
			}
			if "" != outDir {
				out, err := os.Create(playlistFilename)
				if err != nil {
					log.Fatalf("getPlaylist:10> %v \n", err)
				}
				defer out.Close()
				io.Copy(out, buffer)
			}

			// if deleteOld && !firstList {
			// 	cache.RemoveOldest()
			// }
			// firstList = false
			if mpl.Closed {
				close(dlc)
				return
			} else {
				time.Sleep(time.Duration(int64(mpl.TargetDuration * 1000000000)))
			}
		} else {
			log.Printf("getPlaylist:11> Not a valid media playlist")
			if retries == 0 || (retries > 0 && tried >= retries) {
				return
			} else {
				tried += 1
				continue
			}
		}
	}
}

func uri2storagepath(uri string) (path string) {
	path = uri
	 var p []string
	 re1 := regexp.MustCompile("/vds[0-9]+/data[0-9]*/(.*)")
	 p = re1.FindStringSubmatch(uri)
	 // fmt.Printf("re1: %v", p)
	 if p != nil {
	 	path = p[1]
	 	return p[1]
	 }
	 re2 := regexp.MustCompile("/vds[0-9]+/export/data/videos_vod/(.*)")
	 p = re2.FindStringSubmatch(uri)
	 // fmt.Printf("re2: %v", p)
	 if p != nil {
	 	path = p[1]
	 	return p[1]
	 }
	 re3 := regexp.MustCompile("/vds[0-9]+/(v.*)")
	 p = re3.FindStringSubmatch(uri)
	 // fmt.Printf("re3: %v", p)
	 if p != nil {
	 	path = p[1]
	 	return p[1]
	 }
	return uri
}

func downloadMovie(urlStr string, outDir string, recTime time.Duration, retries int,
	useLocalTime bool, skipExists int, callbk func(bool)) {
	// defer func() {
	// 	if x := recover(); x != nil {
	// 		callbk(false)
	// 	} else {
	// 		callbk(true)
	// 	}
	// }()
	msChan := make(chan *Download, 1024)
	go getPlaylist(urlStr, outDir, recTime, retries, useLocalTime, skipExists, msChan)
	downloadSegment(msChan, recTime, retries)
	if callbk != nil {
		callbk(true)
	}
}


func redis_connect(host string, port int, db int) (client *redis.Client, e error) {
	client = redis.New()

	e = client.Connect(host, uint(port))
	if e != nil {
		return
	}
	client.Select(int64(db))
	return
}

func redis_get_indicator(c *redis.Client, k string) (result bool) {
	if c == nil {
		return false
	}
	r, e := c.Get(k + "_indicator")
	// log.Printf("%s:> %s", k+"_indicator", r)
	if e != nil {
		return false
	}
	i, e := strconv.Atoi(r)
	// log.Printf("i=%d, e=%v", i, e)
	if e == nil && i > 0 {
		return true
	}
	return false
}

func redis_get_link(c *redis.Client, k string) (link *string) {
	if c == nil {
		//err = error("Client can not be nil.")
		return nil
	}
	l, _ := c.LPop(k)
	// c.LRange(key, start, stop)
	log.Printf("Get link: %s", l)
	link = &l
	return
}

func redis_set_finished(c *redis.Client, k string, link *string) (err error) {
	err = nil
	if c == nil {
		return
	}
	c.LPush(k+"_finished", *link)
	return
}

func redis_set_failed(c *redis.Client, k string, link *string) (err error) {
	err = nil
	if c == nil {
		return
	}
	c.LPush(k+"failed", *link)
	return
}
