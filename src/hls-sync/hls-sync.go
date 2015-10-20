/*

   This program is free software: you can redistribute it and/or modify
   it under the terms of the GNU General Public License as published by
   the Free Software Foundation, either version 3 of the License, or
   (at your option) any later version.

   This program is distributed in the hope that it will be useful,
   but WITHOUT ANY WARRANTY; without even the implied warranty of
   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
   GNU General Public License for more details.

   You should have received a copy of the GNU General Public License
   along with this program.  If not, see <http://www.gnu.org/licenses/>.

*/

package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"github.com/golang/groupcache/lru"
	"github.com/kz26/m3u8"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path"
	"strings"
	"time"
)

const VERSION = "0.9.5"

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
	// timeStamp  time.Stamp
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
				time.Sleep(time.Duration(int64(mpl.TargetDuration * 1000000000)))
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


func update_status(stc chan *Status, stm map[string] *Status){

	for s := range stc {
		fmt.Println(s.sourceUri)
		stm[s.sourceUri] = s
	}
	
}


func main() {

	duration := flag.Duration("t", time.Duration(0), "Recording duration (0 == infinite)")
	useLocalTime := flag.Bool("l", false, "Use local time to track duration instead of supplied metadata")
	deleteOld := flag.Bool("d", false, "Delete old segments.")
	retryTimes := flag.Int("r", 0, "Retry if failed. 0: Never retry, <0: Keep retry, >0: Retry times.")
	var outputDir string
	flag.StringVar(&outputDir, "o", ".", "Output path. default is '.'")
	var listFilename string
	flag.StringVar(&listFilename, "u", "", "URL list file for multiple sync.")
	flag.StringVar(&USER_AGENT, "ua", fmt.Sprintf("hls-sync/%v", VERSION), "User-Agent for HTTP Client")
	flag.Parse()

	endChan := make(chan bool)

	os.Stderr.Write([]byte(fmt.Sprintf("hls-sync %v - HTTP Live Streaming (HLS) Synchronizar\n", VERSION)))
	os.Stderr.Write([]byte("Copyright (C) 2015 Mingcai SHEN. Licensed for use under the GNU GPL version 3.\n"))

	if flag.NArg() < 1 && listFilename == "" {
		os.Stderr.Write([]byte("Usage: hls-sync [Options] media-playlist-url output-path\n"))
		os.Stderr.Write([]byte("Options:\n"))
		flag.PrintDefaults()
		os.Stderr.Write([]byte("\n"))
		os.Exit(2)
	}

	var linkList []string

	if listFilename != "" {
		f, e := os.Open(listFilename)
		if e != nil {
			fmt.Printf("Open file %s failed! [%v]\n", listFilename, e)
			os.Exit(2)
		}
		defer f.Close()
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			linkList = append(linkList, scanner.Text())
		}
	} else {
		linkList = flag.Args()
	}
	
	stsChan := make(chan *Status, 64)
	stsMap  := make(map[string] *Status, 1024) 

	for _, link := range linkList {
		log.Printf("Start to get link: %s \n", link)
		if !strings.HasPrefix(link, "http") {
			fmt.Println("Media playlist url must begin with http/https")
			os.Exit(2)
		}
		msChan := make(chan *Download, 1024)
		go getPlaylist(link, outputDir, *duration, *deleteOld, *useLocalTime, *retryTimes, msChan)
		go downloadSegment(msChan, *duration)
	}
	
	go update_status(stsChan, stsMap)

	term_c := make(chan os.Signal, 1)

	signal.Notify(term_c, os.Interrupt)
	for {
		select {
		case <-term_c:
			log.Printf("User controled terminated.\n")
			os.Exit(0)
		case <-endChan:
			log.Printf("Sync finished!\n")
			break
		}
	}
}
