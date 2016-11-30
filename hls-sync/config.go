/**
	This source file defines the config options.
 */
package main

import (
	"github.com/BurntSushi/toml"
)

type SyncOption struct {
	// Sync Options ----------------------------------
	Enabled     bool
	Output      string
	Index_Name  string
	Remove_Old  bool
}


type RecordOption struct {
	// Record Options --------------------------------
	Enabled          bool
	Output           string
	Segment_Rewrite  string
	Reindex          bool
	Reindex_Format   string
	Reindex_By       string // hour/minute
}

type SourceOption struct {
	Urls []string
}

type HttpOption struct {
	Enabled bool
	Listen  string   // eg:  tcp://0.0.0.0:8080  or  unix:///tmp/test.sock
	Days    int      // Max shifting days.
	Max     int      // Max length of playlist in minutes.
}

type Option struct {
	// Global Options --------------------------------
	Log_File     string
	Log_Level    string
	Timeout      int
	Retries      int
	User_Agent   string
	Max_Segments int
	Timestamp_type   string // local|program|segment
	Timestamp_Format string
	Timezone_shift   int
	Target_Duration  int
	Program_Time_Format string
	// Sync Option
	Sync         SyncOption
	// Record Option
	Record       RecordOption
	// Source URLs.
	Source       SourceOption
	// Http Service
	Http        HttpOption
}

func LoadConfiguration(filename string, option *Option) (e error) {
	_, e = toml.DecodeFile(filename, option)
	return e
}

