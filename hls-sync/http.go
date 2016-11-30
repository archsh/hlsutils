/***
	This source file contains http service processing.
	HTTP service provides a time-shifting playlist access and generating.
	GET /?playlist={start-timestamp}_{end-timestamp}.m3u8           eg: /?playlist=1479998100_1480004640.m3u8
	GET /?start={start-timestamp}&duration={duration-in-seconds}    eg: /?start=1479998100&duration=6540
	GET /?start={start-timestamp}&end={end-timestamp}               eg: /?start=1479998100&end=1480004640
 */
package main

import (
	"net/http"
	"time"
	log "github.com/Sirupsen/logrus"
	"strconv"
	"fmt"
	"regexp"
	"github.com/archsh/m3u8"
	"net"
	"strings"
	"errors"
)


func (self *Synchronizer) HttpServe() {
	ls := strings.Split(self.option.Http.Listen, "://")
	if len(ls) != 2 {
		log.Errorf("Invalid listen option:> '%s', should use like 'tcp://0.0.0.0:8080' or 'unix:///var/run/test.sock'.", self.option.Http.Listen)
		return
	}
	ln, err := net.Listen(ls[0], ls[1])
	if nil != err {
		log.Errorln("Listen to socket failed:> ", err)
	}
	e := http.Serve(ln, self)
	log.Errorln("HTTP serve failed:> ", e)
}


func (self *Synchronizer) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	_bad_request := func (msg string){
		log.Debugln("Bad Request:> ", msg)
		response.WriteHeader(400)
		response.Header().Set("Content-Type", "text/plain")
		response.Write([]byte(msg))
	}
	if request.Method != "GET" {
		_bad_request("Invalid Request Method!\n")
		return
	}
	playlist := request.URL.Query().Get("playlist")
	start := request.URL.Query().Get("start")
	duration := request.URL.Query().Get("duration")
	end := request.URL.Query().Get("end")
	var _start_time, _end_time time.Time
	if playlist != "" {
		re := regexp.MustCompile("([0-9]+)_([0-9]+).m3u8")
		if !re.MatchString(playlist) {
			_bad_request(fmt.Sprintf("Invalid playlist name format : %s \n",playlist))
			return
		}
		ss := re.FindStringSubmatch(playlist)
		start = ss[1]
		end = ss[2]
	}
	if start != "" {
		_start_sec, e := strconv.ParseInt(start, 10, 64)
		if e != nil {
			_bad_request(fmt.Sprintf("Invalid 'start' parameter: '%s' \n", start))
			return
		}
		_start_time = time.Unix(_start_sec, 0)
		if end != "" {
			_end_sec, e := strconv.ParseInt(end, 10, 64)
			if e != nil {
				_bad_request(fmt.Sprintf("Invalid 'end' parameter: '%s' \n", end))
				return
			}
			_end_time = time.Unix(_end_sec, 0)
		}else if duration != "" {
			_duration_sec, e := strconv.ParseInt(duration, 10, 64)
			if e != nil {
				_bad_request(fmt.Sprintf("Invalid 'duration' parameter: '%s' \n", duration))
				return
			}
			_end_time = _start_time.Add(time.Duration(_duration_sec)*time.Second)
		}else {
			_bad_request("Missing Query Parameter 'duration' or 'end'!\n")
			return
		}
	}else{
		_bad_request("Unknown Query Parameter!\n")
		return
	}
	// Need: Start Timestamp, End Timestamp
	if _start_time.After(_end_time) || _start_time.Equal(_end_time) {
		_bad_request("Start timestamp can not be after end timestamp or as the same as end timestamp.!!!\n")
		return
	}else if time.Now().Sub(_start_time) > time.Duration(self.option.Http.Days*24)*time.Hour {
		_bad_request(fmt.Sprintf("Can not provide shifting before %d days!", self.option.Http.Days))
		return
	}else if _end_time.Sub(_start_time) > time.Duration(self.option.Http.Max)*time.Hour {
		_bad_request(fmt.Sprintf("Can not provide playlist larger than %d hours!", self.option.Http.Max))
		return
	}
	log.Infof("Request Playlist %s -> %s \n", _start_time, _end_time)

	if mpl, e := self.buildPlaylist(_start_time, _end_time); e != nil {
		log.Debugf("Build playlist failed:> %s", e)
		response.WriteHeader(500)
		response.Header().Set("Content-Type", "text/plain")
		response.Write([]byte(fmt.Sprintf("Build playlist failed:> %s", e)))
	}else{
		response.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
		mpl.Encode().WriteTo(response)
	}
}

func (self *Synchronizer) buildPlaylist(start time.Time, end time.Time) (*m3u8.MediaPlaylist, error) {
	return nil, errors.New("Not implemented!")
}