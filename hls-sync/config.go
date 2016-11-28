package main

import (
	//log "github.com/Sirupsen/logrus"
	"github.com/BurntSushi/toml"
)

type SyncOption struct {
	// Sync Options ----------------------------------
	// Output path.
	Enabled     bool
	Output      string
	Output_Temp bool
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
	Timestamp_type   string // local|program|segment
	Timestamp_Format string
	Timezone_shift   int
}

type SourceOption struct {
	Urls []string
}

type Option struct {
	// Global Options --------------------------------
	Log_File   string
	Log_Level  string
	Timeout    int
	Retries    int
	User_Agent string
	Max_Segments int
	// Sync Option
	Sync       SyncOption
	// Record Option
	Record     RecordOption
	// Source URLs.
	Source     SourceOption
}

func LoadConfiguration(filename string, option *Option) (e error) {
	_, e = toml.DecodeFile(filename, option)
	return e
}

