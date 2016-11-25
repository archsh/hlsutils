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
	//log "github.com/Sirupsen/logrus"
	"os"
	//"os/signal"
	//"fmt"
	"flag"
	//"path"
	//"container/list"
)

const VERSION = "0.1.0"

var logging_config = logging.LoggingConfig{Format:logging.DEFAULT_FORMAT, Level:"DEBUG"}

func Usage() {
	guide := `
Scenarios:
  (1) Sync live hls streams from remote hls server.
  (2) Record live streams to local disks.

Usage:
  hls-sync [OPTIONS,...]

Options:
`
	os.Stdout.Write([]byte(guide))
	flag.PrintDefaults()
}


func main() {

	//- `c` string
	//Configuration file instead of command line parameters. Default empty means using parameters.
	//See "Configuration Example" for detail.
	var config string
	flag.StringVar(&config, "c", "", "Configuration file instead of command line parameters. Default empty means using parameters.")
	//- `O` string
	//Output path. A base path for storage segments and play list.
	var output string
	flag.StringVar(&output, "O", ".", "A base path for storage segments and play list.")
	//- `OT` bool
	//Output segment to a temp file first, then rename to target file after finished.
	var output_temp bool
	flag.BoolVar(&output_temp, "OT", false, "Output segment to a temp file first, then rename to target file after finished.")
	//- `SR` string
	//Segment filename rewrite rule. Default empty means simply copy.
	//For example: "%Y/%m/%d/%H/live-%04d.ts"
	var segment_rewrite string
	flag.StringVar(&segment_rewrite, "SR", "", "Segment filename rewrite rule. Default empty means simply copy.")
	//- `TZ` int
	//Timezone shift. Default 0.
	var timezone_shift int
	flag.IntVar(&timezone_shift, "TZ", 0, "Timezone shift. Default 0.")
	//- `TD` float64
	//Target duration in seconds. Default 0.
	var target_duration float64
	flag.Float64Var(&target_duration, "TD", 0, "Target duration in seconds. Default 0.")
	//- `L` string
	//Logging output file. Default 'stdout'.
	var log_file string
	flag.StringVar(&log_file, "L", "", "Logging output file. Default 'stdout'.")
	//- `LV` string
	//Logging level. Default 'INFO'.
	var log_level string
	flag.StringVar(&log_level, "LV", "INFO", "Logging level. ")
	//- `SP` bool
	//Save play list file. Default false.
	var save_playlist bool
	flag.BoolVar(&save_playlist, "SP", false, "Save play list file. Default false.")
	//- `TO` int
	//Request timeout. Default 5.
	var timeout int
	flag.IntVar(&timeout, "TO", 5, "Request timeout. ")
	//- `R` int
	//Retries. Default 1.
	var retries int
	flag.IntVar(&retries, "R", 1, "Retries.")
	//- `UA` string
	//User Agent. Default 'hls-sync v${VERSION}'.
	var user_agent string
	flag.StringVar(&user_agent, "UA", "hls-sync v"+VERSION, "User Agent. ")
	//- `TT` int
	//Timestamp type: 0: local timestamp, 1: program datetime, 2: timestamp from segment filename; default 0.
	var timestamp_type int
	flag.IntVar(&timestamp_type, "TT", 0, "Timestamp type: 0: local timestamp, 1: program datetime, 2: timestamp from segment filename; default 0.")
	//- `ST` string
	//Segment filename timestamp format.
	var segment_time_format string
	flag.StringVar(&segment_time_format, "ST", "", "Segment filename timestamp format.")
	//- `RM` bool
	//Remove old segments.
	var remove_old_segments bool
	flag.BoolVar(&remove_old_segments, "RM", false, "Remove old segments.")
	//- `DP` bool
	//Dump playlist file.
	var dump_playlist bool
	flag.BoolVar(&dump_playlist, "DP", false, "Dump playlist file.")
	//- `PF` string
	//Dumpped playlist filename format.
	var plfile_format string
	flag.StringVar(&plfile_format, "PF", "", "Dumpped playlist filename format.")

	flag.Parse()

	if config != "" {

	}
}
