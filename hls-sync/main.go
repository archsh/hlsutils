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
	"os"
	"flag"
	"fmt"
	"time"
)

const VERSION = "0.9.3"

var logging_config = logging.LoggingConfig{Format:logging.DEFAULT_FORMAT, Level:"DEBUG"}

func Usage() {
	guide := `
Scenarios:
  (1) Sync live hls streams from remote hls server.
  (2) Record live streams to local disks.

Usage:
  hls-sync [OPTIONS,...] [URLs ...]

Options:
`
	os.Stdout.Write([]byte(guide))
	flag.PrintDefaults()
}


func main() {
	option := Option{}
	// Global Arguments ================================================================================================
	//Log_File string
	flag.StringVar(&option.Log_File, "L", "", "Logging output file. Default 'stdout'.")
	//Log_Level string
	flag.StringVar(&option.Log_Level, "V", "INFO", "Logging level. ")
	//Timeout int
	flag.IntVar(&option.Timeout, "T", 5, "Request timeout. ")
	//Retries int
	flag.IntVar(&option.Retries, "R", 1, "Retries.")
	//User_Agent string
	flag.StringVar(&option.User_Agent, "UA", "hls-sync v"+VERSION, "User Agent. ")
	//Max_Segments int
	flag.IntVar(&option.Max_Segments, "MS", 20, "Max segments in playlist.")
	//Timestamp_type string  // local|program|segment
	flag.StringVar(&option.Timestamp_type, "TT", "program", "Timestamp type: local, program, segment.")
	//Timestamp_Format string
	flag.StringVar(&option.Timestamp_Format, "TF", "", "Timestamp format when using timestamp type as 'segment'.")
	//Timezone_shift int
	flag.IntVar(&option.Timezone_shift,"TS", 0, "Timezone shifting by minutes when timestamp is not matching local timezone.")
	//Target_Duration int
	flag.IntVar(&option.Target_Duration, "TD", 0, "Target duration of source. Real target duration will be used when set to 0.")
	//Program_Time_Format string
	flag.StringVar(&option.Program_Time_Format, "PF", time.RFC3339Nano, "To fit some stupid encoders which generated stupid time format.")
	// Sync Arguments ==================================================================================================
	//Enabled bool
	flag.BoolVar(&option.Sync.Enabled, "S", false, "Sync enabled.")
	//Output string
	flag.StringVar(&option.Sync.Output, "SO", ".", "A base path for synced segments and play list.")
	//Index_Name string
	flag.StringVar(&option.Sync.Index_Name, "OI", "live.m3u8", "Index playlist filename.")
	//Remove_Old bool
	flag.BoolVar(&option.Sync.Remove_Old, "RM", false, "Remove old segments.")
	// Record Arguments ================================================================================================
	//Enabled bool
	flag.BoolVar(&option.Record.Enabled, "RC", false, "Record enabled.")
	//Output string
	flag.StringVar(&option.Record.Output, "RO", ".", "Record output path.")
	//Segment_Rewrite string
	flag.StringVar(&option.Record.Segment_Rewrite, "SR", "%Y/%m/%d/%H/live-#:04.ts", "Segment filename rewrite rule. Default empty means simply copy.")
	//Reindex bool
	flag.BoolVar(&option.Record.Reindex, "RI", false, "Re-index playlist when recording.")
	//Reindex_Format string
	flag.StringVar(&option.Record.Reindex_Format, "RF", "%Y/%m/%d/%H/index.m3u8", "Re-index M3U8 filename format.")
	//Reindex_By string // hour/minute
	flag.StringVar(&option.Record.Reindex_By, "RB", "hour", "Re-index by 'hour' or 'minute'.")
	// Functional Arguments ============================================================================================
	var config string
	flag.StringVar(&config, "c", "", "Configuration file instead of command line parameters. Default empty means using parameters.")
	var check bool
	flag.BoolVar(&check, "C", false, "Check options.")
	var showVersion bool
	flag.BoolVar(&showVersion, "v", false, "Display version info.")
	flag.Parse()


	if showVersion {
		os.Stderr.Write([]byte(fmt.Sprintf("hls-sync v%v\n", VERSION)))
		os.Exit(0)
	}
	os.Stderr.Write([]byte(fmt.Sprintf("hls-sync v%v - HTTP Live Streaming (HLS) Synchronizer.\n", VERSION)))
	os.Stderr.Write([]byte("Copyright (C) 2015 Mingcai SHEN <archsh@gmail.com>. Licensed for use under the GNU GPL version 3.\n"))
	if config != "" {

		if e := LoadConfiguration(config, &option); e != nil {
			os.Stderr.Write([]byte(fmt.Sprintf("Load config<%s> failed: %s.\n", config, e)))
			os.Exit(1)
		}else{
			os.Stderr.Write([]byte(fmt.Sprintf("Loaded config from <%s>.\n", config)))
		}
		if flag.NArg() > 0 {
			option.Source.Urls = append(option.Source.Urls, flag.Args()...)
		}
	}else{
		if flag.NArg() < 1 {
			os.Stderr.Write([]byte("!!! At least one source URL is required!\n"))
			Usage()
			os.Exit(1)
		}else{
			option.Source.Urls = flag.Args()
		}
	}
	if check {
		os.Stderr.Write([]byte(fmt.Sprint("Checking options ...\n")))
		os.Stderr.Write([]byte(fmt.Sprintf("Options> \n %+v \n", option)))
		os.Exit(0)
	}

	logging_config.Filename = option.Log_File
	logging_config.Level = option.Log_Level
	if option.Log_File != "" {
		logging.InitializeLogging(&logging_config, false, logging_config.Level)
	}else{
		logging.InitializeLogging(&logging_config, true, logging_config.Level)
	}
	defer logging.DeinitializeLogging()
	//os.Stderr.Write([]byte(fmt.Sprintf(" %v \n", option)))
	if sync, e := NewSynchronizer(&option); e != nil {
		os.Stderr.Write([]byte(fmt.Sprintf("Start failed: %s.\n", e)))
		os.Exit(1)
	}else{
		sync.Run()
	}
}
