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
	"github.com/archsh/hlsutils/helpers/logging"
	log "github.com/Sirupsen/logrus"
	"os"
	"os/signal"
	"fmt"
	"flag"
)

const VERSION = "0.9.8-dev"

var logging_config = logging.LoggingConfig{Format:logging.DEFAULT_FORMAT, Level:"DEBUG"}

func Usage() {
	guide := `
Scenarios:
  (1) Sync live hls streams from remote hls server.
  (2) Record live streams to local disks.

Usage:
  hls-get [OPTIONS,...] [URL1,URL2,...]

Options:
`
	os.Stdout.Write([]byte(guide))
	flag.PrintDefaults()
}


func main() {

	//O  'output'     - [STRING] Output directory. Default '.'.
	var output string
	flag.StringVar(&output, "O", ".", "Output base directory.")
	//SR 'sync_rewrite'    - [STRING] Rewrite sync output path method. Default empty means simple copy.
	var sync_rewrite string
	flag.StringVar(&sync_rewrite, "SR", "", "Rewrite sync output path method. Empty means simple copy.")
	//RR 'record_rewrite'     - [STRING] Rewrite record output method. Default empty means simple copy.
	var record_rewrite string
	flag.StringVar(&record_rewrite, "RR", "", "Rewrite record output method. Empty means simple copy.")
	//UA 'user_agent'    - [STRING] UserAgent. Default is 'hls-get' with version num.
	var user_agent string
	flag.StringVar(&user_agent, "UA", "hls-get v" + VERSION, "UserAgent.")
	//L  'log'   - [STRING] Logging output file. Default 'stdout'.
	var log_file string
	flag.StringVar(&log_file, "L", "", "Logging output file. Default 'stdout'.")
	//R  'retry' - [INTEGER] Retry times if download fails.
	var retries int
	flag.IntVar(&retries, "R", 0, "Retry times if download fails.")
	//RS 'redirect'   - [STRING] Redirect server request.
	var redirect string
	flag.StringVar(&redirect, "RS", "", "Redirect server request.")
	//TO 'timeout'    - [INTEGER] Request timeout in seconds.
	var timeout int
	flag.IntVar(&timeout, "TO", 20, "Request timeout in seconds.")
	//HL 'hlslink'     - [STRING] HLS Links filename.
	var hlslink string
	flag.StringVar(&hlslink, "HL", "", "HLS Links filename.")
	//UL 'use_localtime'
	var use_localtime bool
	flag.BoolVar(&use_localtime, "UL", false, "Use local time to track duration instead of supplied metadata.")
	//US `use_segment_time'
	var use_segment_time bool
	flag.BoolVar(&use_segment_time, "US", false, "Use segment timestamp. Please specify the timestamp format.")
	//ST `segment_timestamp` // 1-20160922T022309-67294.ts
	var segment_timestamp string
	flag.StringVar(&segment_timestamp, "ST", "1-%Y%M%DT%H%m%s-xxx.ts", "Segment time format.")
	//SD `segment_duration`
	//var segment_duration int
	//flag.IntVar(&segment_duration, "SD", 5, "Segment duration in seconds.")

	flag.Parse()

	if !use_localtime && !use_segment_time {
		use_localtime = true
	}
	if use_localtime && use_segment_time {
		use_segment_time = true
	}

	default_option := Option{}
	default_option.Record_output = output
	default_option.Sync_output = output
	default_option.Sync_rewrite = sync_rewrite
	default_option.Record_rewrite = record_rewrite
	default_option.Redirect = redirect
	default_option.Timeout = timeout
	default_option.Use_localtime = use_localtime
	default_option.Use_segment_time = use_segment_time
	default_option.User_agent = user_agent
	//default_option.Segment_duration = segment_duration
	default_option.Segment_timestamp = segment_timestamp


	var dump_flags = func () {
		fmt.Println("=================================== Args =================================")
		fmt.Println(">>", "output:", output)
		fmt.Println(">>", "sync_rewrite:", sync_rewrite)
		fmt.Println(">>", "record_rewrite:", record_rewrite)
		fmt.Println(">>", "user_agent:", user_agent)
		fmt.Println(">>", "log_file:", log_file)
		fmt.Println(">>", "retries:", retries)
		fmt.Println(">>", "redirect:", redirect)
		fmt.Println(">>", "timeout:", timeout)
		fmt.Println(">>", "hlslink:", hlslink)
		fmt.Println(">>", "use_segment_time:", use_segment_time)
		fmt.Println(">>", "segment_timestamp:", segment_timestamp)
		//fmt.Println(">>", "segment_duration:", segment_duration)
		fmt.Println("==========================================================================")
		fmt.Printf("%+v \n", default_option)
		fmt.Println("==========================================================================")
	}

	os.Stderr.Write([]byte(fmt.Sprintf("hls-sync v%v - HTTP Live Streaming (HLS) Synchronizer.\n", VERSION)))
	os.Stderr.Write([]byte("Copyright (C) 2015 Mingcai SHEN <archsh@gmail.com>. Licensed for use under the GNU GPL version 3.\n"))
	var hls_links []*SyncOption
	if hlslink != "" {
		hls_links = Load_HLS_Links(hlslink)
	}else if flag.NArg() > 0 {
		for _, ll := range flag.Args() {
			op := Option{Source:ll}
			so, e := Build_Sync_Option(&op, &default_option)
			if nil != e {
				log.Errorln("Failed to build sync option:", e)
				continue
			}
			hls_links = append(hls_links, so)
		}
	}

	dump_flags()

	if len(hls_links) < 1 {
		Usage()
		return
	}

	logging_config.Filename = log_file
	if log_file != "" {
		logging.InitializeLogging(&logging_config, false, logging_config.Level)
	}else{
		logging.InitializeLogging(&logging_config, true, logging_config.Level)
	}
	defer logging.DeinitializeLogging()



	for _, lk := range hls_links {
		hs := NewHLSSynchronizer(lk)
		go hs.Run()
	}

	endChan := make(chan bool)
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
