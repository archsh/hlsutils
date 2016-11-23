package main

import (
	"bytes"
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
	"errors"
)

type Download struct {
	URI           string
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
	var totalSuccess int64
	var totalFailed int64
	totalDownloaded = 0
	totalFailed = 0
	for {
		if self._total > 0 && totalDownloaded >= self._total {
			log.Infoln("Reache total of downloads:", self._total)
			break;
		}
		var num int
		if self._total > 0 && self._concurrent > int(self._total - totalDownloaded) {
			num = int(self._total - totalDownloaded)
		}else{
			num = self._concurrent
		}
		urls, err := self._dl_intf.NextLinks(num)
		//log.Debugln("length of urls:", len(urls))
		if nil != err || len(urls)==0 {
			log.Errorln("Can not get links!", err)
			break;
		}
		var wg sync.WaitGroup
		wg.Add(len(urls))
		for _, l := range urls {
			log.Debugln(" Downloading ", l, "...")
			go func (lk string) {
				self.Download(lk, self._output, "", func (url string, dest string, ret_code int, ret_msg string){
					if ret_code == 0 {
						totalSuccess += 1
					}else{
						totalFailed += 1
					}
					totalDownloaded += 1
					self._dl_intf.SubmitResult(url, dest, ret_code, ret_msg)
				})
				wg.Done()
			}(l)
		}
		wg.Wait()
		if len(urls) < self._concurrent || len(urls) < 1 {
			log.Infoln("End of download list.")
			break;
		}

	}
	log.Infoln("Total Downloaded:", totalDownloaded)
	log.Infoln("Total Success:", totalSuccess)
	log.Infoln("Total Failed:", totalFailed)
}

func (self *HLSGetter) doRequest(req *http.Request) (*http.Response, error) {
	req.Header.Set("User-Agent", self._user_agent)
	req.Header.Add("Accept-Encoding", "identity")
	resp, err := self._client.Do(req)
	return resp, err
}

func (self *HLSGetter) GetSegment(url string, filename string, skip_exists bool, retries int) (string, error) {
	var out *os.File
	if skip_exists && exists(filename) {
		log.Infoln("Segment exists: ", filename)
		return filename, nil
	}
	if retries < 1 {
		retries = 1
	}
	for i:=0; i< retries; i++ {
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			log.Errorf("GetSegment:1> Create new request failed: %v\n", err)
			return filename, err
		}

		if "" != filename {
			err = os.MkdirAll(filepath.Dir(filename), 0777)
			if err != nil {
				log.Errorf("GetSegment:2> Create path %s failed :%v\n", filepath.Dir(filename), err)
				return filename, err
			}
			out, err = os.Create(filename)
		} else {
			out, err = ioutil.TempFile("./", "__savedTempSegment")
		}
		if err != nil {
			log.Errorf("GetSegment:3> Create file %s failed: %v\n", filename, err)
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
			log.Errorf("GetSegment:4> do request failed: %v\n", err)
			time.Sleep(time.Duration(1) * time.Second)
			continue
		}
		if resp.StatusCode != 200 {
			log.Errorf("Received HTTP %v for %v \n", resp.StatusCode, url)
			time.Sleep(time.Duration(1) * time.Second)
			continue
		}
		_, err = io.Copy(out, resp.Body)
		if err != nil {
			log.Errorf("GetSegment:5> Copy response body failed: %v\n", err)
			time.Sleep(time.Duration(1) * time.Second)
			continue
		}
		resp.Body.Close()
		log.Debugf("Downloaded %v into %v\n", url, filename)
		return filename, nil
	}

	return "", errors.New("Failed to download segment!")
}

func (self *HLSGetter) GetPlaylist(urlStr string, outDir string, filename string, retries int, skip_exists bool) (segments []*Download, dest string, ret_code int, ret_msg string) {
	if retries < 1 {
		retries = 1
	}
	for i:=0; i< retries; i++ {
		if self._redirect_url != "" {
			urlStr = fmt.Sprintf(self._redirect_url,urlStr)
		}
		log.Debugln("GetPlaylist:> Get ", urlStr)
		req, err := http.NewRequest("GET", urlStr, nil)
		if err != nil {
			log.Errorf("GetPlaylist:> Failed to build request: %v \n", err)
			continue
		}
		resp, err := self.doRequest(req)
		if err != nil {
			log.Errorln("GetPlaylist:> Request failed:", err)
			time.Sleep(time.Duration(1) * time.Second)
			continue
		}
		if filename == "" {
			filename = self.PathRewrite(resp.Request.URL.Path)
		}
		respBody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Errorf("GetPlaylist:> Read response failed: %v \n", err)
			time.Sleep(time.Duration(1) * time.Second)
			continue
		}
		buffer := bytes.NewBuffer(respBody)
		playlistFilename := filepath.Join(outDir, filename)
		err = os.MkdirAll(filepath.Dir(playlistFilename), 0777)
		if err != nil {
			log.Errorf("GetPlaylist:> Create directory '%s' failed: %v \n", filepath.Dir(playlistFilename), err)
			ret_code = -1
			ret_msg = fmt.Sprintf("Create directory '%s' failed: %v \n", filepath.Dir(playlistFilename), err)
			return
		}
		playlist, listType, err := m3u8.Decode(*buffer, true)
		if err != nil {
			log.Errorf("GetPlaylist:> Decode M3U8 failed: %v \n", err)
			continue
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
					msURI, _ = url.QueryUnescape(v.URI)
				} else {
					msUrl, _ := resp.Request.URL.Parse(v.URI)
					msURI, _ = url.QueryUnescape(msUrl.String())
				}
				segname := self.SegmentRewrite(v.URI,idx)  //fmt.Sprintf("%04d.ts", idx)
				msFilename = filepath.Join(filepath.Dir(playlistFilename), segname)
				//mpl.Segments[idx].URI = segname
				new_mpl.Append(segname, v.Duration, v.Title)
				segments = append(segments, &Download{msURI, msFilename, urlStr, uint(segs), uint(idx + 1), 0})
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
			//dest = playlistFilename
			return segments, playlistFilename, 0, ""
		} else {
			log.Errorln("GetPlaylist:11> Not a valid media playlist")
			time.Sleep(time.Duration(1) * time.Second)
		}
	}
	return nil, "", -1, "Failed to get playlist."
}

func (self *HLSGetter) Download(urlStr string, outDir string, filename string, callback func(url string, dest string, ret_code int, ret_msg string)){
	var	dest string
	var	ret_code int
	var ret_msg string
	var segments []*Download
	failures := 0
	log.Debugln("Download> ", urlStr, outDir)
	segments, dest, ret_code, ret_msg = self.GetPlaylist(urlStr, outDir, filename, self._retries, self._skip_exists)
	if len(segments) < 1 || ret_code != 0 {
		callback(urlStr, dest, ret_code, ret_msg)
		return
	}

	for i:=0; i<len(segments); i+= self._concurrent {
		var segs []*Download
		if i+self._concurrent < len(segments) {
			segs = segments[i:i+self._concurrent]
		}else{
			segs = segments[i:]
		}
		var wg sync.WaitGroup
		wg.Add(len(segs))
		for _, seg := range segs {
			//log.Debugln(">>> Seg:", seg.URI)
			go func (ps *Download) {
				s, e := self.GetSegment(ps.URI, ps.Filename, self._skip_exists, self._retries)
				if e != nil {
					failures += 1
					log.Errorln("Download Segment failed:", ps.URI, e)
				}else{
					log.Debugln("Dowloaded >", s)
				}
				wg.Done()
			}(seg)
		}
		wg.Wait()
	}

	if failures > 1 {
		callback(urlStr, dest, -9, fmt.Sprintf("Failed to download %d segments!", failures))
	}else{
		callback(urlStr, dest, 0, "")
	}
}

