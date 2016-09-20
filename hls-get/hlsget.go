package main

import (
	"bytes"
	"github.com/golang/groupcache/lru"
	"github.com/kz26/m3u8"
	"io"
	"io/ioutil"
	log "github.com/Sirupsen/logrus"
	"net/http"
	"net/url"
	"os"
	//"path"
	//"regexp"
	"strings"
	"time"
	"path/filepath"
	"fmt"
	"sync"
)

type Download struct {
	URI           string
	totalDuration time.Duration
	Filename      string
	refer         string
	totalSegments uint
	index         uint
	retries       int
}

type HLSGetter struct {
	_client  *http.Client
	_dl_intf  DL_Interface
	_path_rewriter    StringRewriter
	_segment_rewriter StringRewriter

	_output string
	_retries int
	_timeout int
	_skip_exists bool
	_user_agent string
	_concurrent int
	_redirect_url string
	_total int64
}

func NewHLSGetter(dl_intf DL_Interface, output string,
                  path_rewriter StringRewriter, segment_rewriter StringRewriter,
                  retries int, timeout int, skip_exists bool, redirect string, concurrent int, total int64) *HLSGetter {
	hls := new(HLSGetter)
	hls._client = &http.Client{Timeout: time.Duration(timeout)*time.Second}
	hls._dl_intf = dl_intf
	hls._output = output
	hls._path_rewriter = path_rewriter
	hls._segment_rewriter = segment_rewriter
	hls._redirect_url = redirect
	hls._retries = retries
	hls._timeout = timeout
	hls._skip_exists = skip_exists
	hls._concurrent = concurrent
	hls._user_agent = "hls-get v"+VERSION
	hls._total = total
	return hls
}

func (self *HLSGetter) SetUA(ua string) {
	self._user_agent = ua
}

func (self *HLSGetter) PathRewrite(intput string) string {
	if self._path_rewriter != nil {
		return self._path_rewriter.RunString(intput)
	}
	return intput
}

func (self *HLSGetter) SegmentRewrite(input string, idx int) string {
	if self._segment_rewriter != nil {
		return self._segment_rewriter.RunString(input)
	}
	return input
}

func (self *HLSGetter) Run() {
	if self._concurrent < 1 {
		log.Fatalln("Concurrent can not less than 1!")
	}
	if self._dl_intf == nil {
		log.Fatalln("Download List Interface can not be nil!")
	}
	var totalDownloaded int64
	var totalFailed int64
	totalDownloaded = 0
	totalFailed = 0
	for {
		if self._total > 0 && totalDownloaded >= self._total {
			log.Infoln("Reache total of downloads:", self._total)
			break;
		}
		urls, err := self._dl_intf.NextLinks(self._concurrent)
		//log.Debugln("length of urls:", len(urls))
		if nil != err || len(urls)==0 {
			log.Errorln("Can not get links!", err)
			break;
		}
		var wg sync.WaitGroup
		wg.Add(len(urls))
		for _, l := range urls {
			log.Debugln(" Downloading ", l, "...")
			go func () {
				self.Download(l, self._output, func (url string, dest string, ret_code int, ret_msg string){
					if ret_code == 0 {
						totalDownloaded += 1
					}else{
						totalFailed += 1
					}
					self._dl_intf.SubmitResult(url, dest, ret_code, ret_msg)
				})
				wg.Done()
			}()
		}
		wg.Wait()
		//log.Debugln("length of urls:", len(urls), self._concurrent)
		if len(urls) < self._concurrent || len(urls) < 1 {
			log.Infoln("End of download list.")
			break;
		}

	}
	log.Infoln("Total Downloaded:", totalDownloaded)
	log.Infoln("Total Failed:", totalFailed)
}

func (self *HLSGetter) doRequest(req *http.Request) (*http.Response, error) {
	req.Header.Set("User-Agent", self._user_agent)
	resp, err := self._client.Do(req)
	return resp, err
}

func (self *HLSGetter) GetSaveSegment(url string, filename string) (string, error) {
	var out *os.File
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Errorf("getsaveSegment:1> Create new request failed: %v\n", err)
		return filename, err
	}

	if "" != filename {
		err = os.MkdirAll(filepath.Dir(filename), 0777)
		if err != nil {
			log.Errorf("getsaveSegment:2> Create path %s failed :%v\n", filepath.Dir(filename), err)
			return filename, err
		}

		out, err = os.Create(filename)
	} else {
		out, err = ioutil.TempFile("./", "__savedTempSegment")
	}
	if err != nil {
		log.Errorf("getsaveSegment:3> Create file %s failed: %v\n", filename, err)
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

	resp, err := self.doRequest(req)
	if err != nil {
		log.Errorf("getsaveSegment:4> do request failed: %v\n", err)
		return filename, err
	}
	if resp.StatusCode != 200 {
		log.Errorf("Received HTTP %v for %v \n", resp.StatusCode, url)
		return filename, err
	}
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		log.Errorf("getsaveSegment:5> Copy response body failed: %v\n", err)
		return filename, err
	}
	resp.Body.Close()
	log.Debugf("Downloaded %v into %v\n", url, filename)
	return filename, err
}

func (self *HLSGetter) GetPlaylist(dlc chan *Download, urlStr string, outDir string, retries int, skip_exists bool) (link string, dest string, ret_code int, ret_msg string){
	startTime := time.Now()
	var playlistFilename string
	var recDuration time.Duration = 0
	var tried = 0
	cache := lru.New(1024)
	link = urlStr
	log.Debugf("URI: %s, output: %s \n", urlStr, outDir)
	if "" != outDir {
		err := os.MkdirAll(outDir, 0755)
		if nil != err {
			log.Errorln("Failed to create directory:", err)
			ret_code = -1
			ret_msg = fmt.Sprintf("%v", err)
			return
		}
		outPath, err := os.Open(outDir)
		if err != nil {
			log.Errorf("GetPlaylist:1> %v \n", err)
			ret_code = -1
			ret_msg = fmt.Sprintf("%v", err)
			return
		}
		defer outPath.Close()
		fstat, err := outPath.Stat()
		if err != nil {
			log.Errorf("GetPlaylist:2> %v \n", err)
			ret_code = -1
			ret_msg = fmt.Sprintf("%v", err)
			return
		}
		if fstat.IsDir() != true {
			log.Errorln("GetPlaylist:3> Output is not a directory!")
			ret_code = -1
			ret_msg = fmt.Sprintf("'%s' is not a directory!", outDir)
			return
		}
	}

	for {
		if self._redirect_url != "" {
			urlStr = fmt.Sprintf(self._redirect_url,urlStr)
		}
		req, err := http.NewRequest("GET", urlStr, nil)
		if err != nil {
			log.Errorf("GetPlaylist:4> %v \n", err)
			if retries == 0 || (retries > 0 && tried >= retries) {
				ret_code = -2
				ret_msg = fmt.Sprintf("Reached maximum retry. [%d]", tried)
				return
			} else {
				tried += 1
				continue
			}
		}
		resp, err := self.doRequest(req)
		if err != nil {
			log.Errorln("GetPlaylist:> ", err)
			time.Sleep(time.Duration(3) * time.Second)
			if retries == 0 || (retries > 0 && tried >= retries) {
				ret_code = -2
				ret_msg = fmt.Sprintf("Reached maximum retry. [%d]", tried)
				return
			} else {
				tried += 1
				continue
			}
		}
		filename := self.PathRewrite(resp.Request.URL.Path)
		respBody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Errorf("GetPlaylist:5> %v \n", err)
			if retries == 0 || (retries > 0 && tried >= retries) {
				ret_code = -2
				ret_msg = fmt.Sprintf("Reached maximum retry. [%d]", tried)
				return
			} else {
				tried += 1
				continue
			}
		}
		buffer := bytes.NewBuffer(respBody)
		playlistFilename = filepath.Join(outDir, filename)
		err = os.MkdirAll(filepath.Dir(playlistFilename), 0777)
		if err != nil {
			log.Errorf("GetPlaylist:6> %v \n", err)
			ret_code = -1
			ret_msg = fmt.Sprintf("%v", err)
			return
		}

		playlist, listType, err := m3u8.Decode(*buffer, true)
		if err != nil {
			log.Errorf("GetPlaylist:7> %v \n", err)
			if retries == 0 || (retries > 0 && tried >= retries) {
				ret_code = -3
				ret_msg = fmt.Sprintf("%v", err)
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
			new_mpl, err := m3u8.NewMediaPlaylist(uint(segs), uint(segs))
			for idx, v := range mpl.Segments {
				if v == nil {
					continue
				}
				var msURI string
				var msFilename string
				if strings.HasPrefix(v.URI, "http") {
					msURI, err = url.QueryUnescape(v.URI)
					if err != nil {
						log.Errorf("getPlaylist:8> %v \n", err)
						if retries == 0 || (retries > 0 && tried >= retries) {
							ret_code = -3
							ret_msg = fmt.Sprintf("%v", err)
							return
						} else {
							tried += 1
							continue
						}
					}
				} else {
					msUrl, err := resp.Request.URL.Parse(v.URI)
					if err != nil {
						log.Print(err)
						continue
					}
					msURI, err = url.QueryUnescape(msUrl.String())
					if err != nil {
						log.Errorf("GetPlaylist:9> %v \n", err)
						if retries == 0 || (retries > 0 && tried >= retries) {
							ret_code = -3
							ret_msg = fmt.Sprintf("%v", err)
							return
						} else {
							tried += 1
							continue
						}
					}
					//msFilename = filepath.Join(outDir, uri2storagepath(msUrl.Path))
				}
				segname := self.SegmentRewrite(v.URI,idx)  //fmt.Sprintf("%04d.ts", idx)
				msFilename = filepath.Join(filepath.Dir(playlistFilename), segname)
				//mpl.Segments[idx].URI = segname
				new_mpl.Append(segname, v.Duration, v.Title)
				//log.Infof("Appended segment[%d]=%s\n", idx, segname)
				//seg := v
				//seg.URI = segname
				//new_mpl.Segments = append(new_mpl.Segments, seg)
				_, hit := cache.Get(msURI)
				if skip_exists && exists(msFilename) {
					log.Debugf("Segment [%s] exists!", msFilename)
				} else if !hit {
					cache.Add(msURI, msFilename)
					if false {
						recDuration = time.Now().Sub(startTime)
					} else {
						recDuration += time.Duration(int64(v.Duration * 1000000000))
					}
					if "" == outDir {
						msFilename = ""
					}
					dlc <- &Download{msURI, recDuration, msFilename, urlStr, uint(segs), uint(idx + 1), 0}
				}
				//if recTime != 0 && recDuration != 0 && recDuration >= recTime {
				//	close(dlc)
				//	return
				//}
			}
			log.Debugln("GetPlaylist> Writing playlist to ", playlistFilename, "...")
			out, err := os.Create(playlistFilename)
			if err != nil {
				log.Errorf("GetPlaylist:10> %v \n", err)
				ret_code = -3
				ret_msg = fmt.Sprint(err)
				return
			}
			defer out.Close()
			new_mpl.Close()
			buf := new_mpl.Encode()
			io.Copy(out, buf)
			if mpl.Closed {
				close(dlc)
			} else {
				time.Sleep(time.Duration(int64(mpl.TargetDuration * 1000000000)))
			}
			dest = playlistFilename
			return
		} else {
			log.Errorln("GetPlaylist:11> Not a valid media playlist")
			if retries == 0 || (retries > 0 && tried >= retries) {
				ret_code = -3
				ret_msg = fmt.Sprint("Not a valid media playlist.")
				return
			} else {
				tried += 1
				continue
			}
		}
	}

	return
}

func (self *HLSGetter) GetSegments(segChan chan *Download, retries int) (int, string) {
	for v := range segChan {
		RETRY:
		log.Infof("downloadSegment: %s \n", v.Filename)
		_, err := self.GetSaveSegment(v.URI, v.Filename)
		if err != nil {
			log.Errorf("downloadSegment:> %v \n", err)
			if retries < 0 || (retries > 0 && v.retries < retries) {
				v.retries += 1
				log.Debugf("downloadSegment:> Retry to download %s in %d times. \n", v.URI, v.retries)
				time.Sleep(time.Duration(3) * time.Second)
				goto RETRY
			}else{
				return -5, fmt.Sprint(err)
			}
		}
	}
	return 0, ""
}

func (self *HLSGetter) Download(urlStr string, outDir string, callback func(url string, dest string, ret_code int, ret_msg string)){
	msChan := make(chan *Download, 1024)
	//recTime := 1 * time.Second
	var url string
	var	dest string
	var	ret_code int
	var ret_msg string
	go func (){
		url, dest, ret_code, ret_msg = self.GetPlaylist(msChan, urlStr, outDir, self._retries, self._skip_exists)
	}()
	ret_code, ret_msg = self.GetSegments(msChan, self._retries)
	if callback != nil {
		callback(url, dest, ret_code, ret_msg)
	}
}

