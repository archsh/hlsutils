/*
 * hls-get
 */

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

import "flag"
import "fmt"
import "io"
import "io/ioutil"
import "net/http"
import "net/url"
import "log"
import "os"
import "path"
import "time"
import "bytes"
import "strconv"
import "github.com/golang/groupcache/lru"
import "strings"
import "github.com/kz26/m3u8"
import "github.com/gosexy/redis"

const VERSION = "0.9.3"

var USER_AGENT string

var client = &http.Client{}

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
}

func getsaveSegment(url string, filename string) (string, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatal(err)
	}

	err = os.MkdirAll(path.Dir(filename), 0777)
	if err != nil {
		log.Fatal(err)
	}

	out, err := os.Create(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer out.Close()
	resp, err := doRequest(client, req)
	if err != nil {
		log.Fatal(err)
	}
	if resp.StatusCode != 200 {
		log.Printf("Received HTTP %v for %v \n", resp.StatusCode, url)
	}
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	resp.Body.Close()
	log.Printf("Downloaded %v into %v\n", url, filename)
	return filename, err
}

func downloadSegment(dlc chan *Download, recTime time.Duration) {
	for v := range dlc {
		log.Printf("downloadSegment: %v", v)
		_, err := getsaveSegment(v.URI, v.Filename)
		if err != nil {
			log.Fatal(err)
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

func getPlaylist(urlStr string, outDir string, recTime time.Duration, deleteOld bool, useLocalTime bool, skipExists bool, dlc chan *Download) {
	startTime := time.Now()
	var recDuration time.Duration = 0
	var firstList = true
	cache := lru.New(1024)
	cache.OnEvicted = func(key lru.Key, value interface{}) {
		fname, res := value.(string)
		if res {
			deleteOldSegment(fname)
		}
	}
	outPath, err := os.Open(outDir)
	if err != nil {
		log.Fatal(err)
	}
	defer outPath.Close()
	fstat, err := outPath.Stat()
	if err != nil {
		log.Fatal(err)
	}
	if fstat.IsDir() != true {
		log.Fatal("Output is not a directory!")
	}

	//	playlistUrl, err := url.Parse(urlStr)
	//	if err != nil {
	//		log.Fatal(err)
	//	}

	for {
		req, err := http.NewRequest("GET", urlStr, nil)
		if err != nil {
			log.Fatal(err)
		}
		resp, err := doRequest(client, req)
		if err != nil {
			log.Print(err)
			time.Sleep(time.Duration(3) * time.Second)
		}
		respBody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
		}
		buffer := bytes.NewBuffer(respBody)
		playlistFilename := path.Join(outDir, resp.Request.URL.Path)
		err = os.MkdirAll(path.Dir(playlistFilename), 0777)
		if err != nil {
			log.Fatal(err)
		}
		playlist, listType, err := m3u8.Decode(*buffer, true)
		//		m3u8.DecodeFrom(resp.Body, true)
		if err != nil {
			log.Fatal(err)
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
							log.Fatal(err)
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
							log.Fatal(err)
						}
						msFilename = path.Join(outDir, msUrl.Path)
					}
					_, hit := cache.Get(msURI)
					if !hit && (!skipExists || !exists(msFilename)) {
						cache.Add(msURI, msFilename)
						if useLocalTime {
							recDuration = time.Now().Sub(startTime)
						} else {
							recDuration += time.Duration(int64(v.Duration * 1000000000))
						}
						dlc <- &Download{msURI, recDuration, msFilename, urlStr, uint(segs), uint(idx + 1)}
					}
					if recTime != 0 && recDuration != 0 && recDuration >= recTime {
						close(dlc)
						return
					}
				}
			}
			out, err := os.Create(playlistFilename)
			if err != nil {
				log.Fatal(err)
			}
			io.Copy(out, buffer)
			out.Close()
			if deleteOld && !firstList {
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
			log.Fatal("Not a valid media playlist")
		}
	}
}

func main() {

	duration := flag.Duration("t", time.Duration(0), "Recording duration (0 == infinite)")
	useLocalTime := flag.Bool("l", false, "Use local time to track duration instead of supplied metadata")
	deleteOld := flag.Bool("r", false, "Remove old segments.")
	var output string
	flag.StringVar(&output, "o", "./", "Output path for sync files.")
	flag.StringVar(&USER_AGENT, "ua", fmt.Sprintf("hls-sync/%v", VERSION), "User-Agent for HTTP Client")
	var redisHost string
	flag.StringVar(&redisHost, "h", "", "Redis server hostname or IP address.")
	redisPort := flag.Int("p", 6379, "Redis server port number, default is 6379.")
	redisDb := flag.Int("d", 0, "Redis db number, default 0.")
	skipExists := flag.Bool("s", false, "Skip exists files.")
	var redisKey string
	flag.StringVar(&redisKey, "k", "DOWNLOAD_MOVIES", "The base list key name in redis.")
	flag.Parse()

	os.Stderr.Write([]byte(fmt.Sprintf("hls-sync %v - HTTP Live Streaming (HLS) Synchronizer\n", VERSION)))
	os.Stderr.Write([]byte("Copyright (C) 2015 Mingcai SHEN. Licensed for use under the GNU GPL version 3.\n"))

	if redisHost == "" && flag.NArg() < 1 {
		os.Stderr.Write([]byte("Usage: hls-sync [Options] media-playlist-url output-path\n"))
		os.Stderr.Write([]byte("Options:\n"))
		// os.Stderr.Write([]byte("  -l=bool \n"))
		// os.Stderr.Write([]byte("  -t duration\n"))
		// os.Stderr.Write([]byte("  -ua user-agent\n"))
		// os.Stderr.Write([]byte("  -d=bool delete old segments.\n"))
		// os.Stderr.Write([]byte("\n"))
		flag.PrintDefaults()
		os.Stderr.Write([]byte("\n"))
		os.Exit(2)
	}

	if redisHost != "" {
		log.Println("Using redis as task dispatcher...")
		rc, err := redis_connect(redisHost, *redisPort, *redisDb)
		if err != nil {
			log.Fatal("Can not connect to redis server.")
		}
		for {
			indicator := redis_get_indicator(rc, redisKey)
			if !indicator {
				log.Println("Indicator not activated!")
				time.Sleep(10 * time.Second)
				continue
			}
			link := redis_get_link(rc, redisKey)
			if link == nil || *link == "" {
				log.Println("No download link!")
				time.Sleep(5 * time.Second)
				continue
			}
			msChan := make(chan *Download, 1024)
			go getPlaylist(*link, output, *duration, *deleteOld, *useLocalTime, *skipExists, msChan)
			downloadSegment(msChan, *duration)
			redis_set_finished(rc, redisKey, *link)

		}
	} else {
		msChan := make(chan *Download, 1024)
		for i := 0; i < flag.NArg(); i++ {
			if !strings.HasPrefix(flag.Arg(i), "http") {
				log.Fatal("Media playlist url must begin with http/https")
			}
			go getPlaylist(flag.Arg(i), output, *duration, *deleteOld, *useLocalTime, *skipExists, msChan)
		}
		downloadSegment(msChan, *duration)
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
	log.Printf("Get link: %s", l)
	link = &l
	return
}

func redis_set_finished(c *redis.Client, k string, link string) (err error) {
	err = nil
	if c == nil {
		return
	}
	c.LPush(k+"_finished", &link)
	return
}

func redis_set_failed(c *redis.Client, k string, link string) (err error) {
	err = nil
	if c == nil {
		return
	}
	c.LPush(k+"failed", &link)
	return
}
