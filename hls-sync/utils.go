package main

import (
	log "github.com/Sirupsen/logrus"
	"github.com/BurntSushi/toml"
)

type Option struct {
	//source=http://1.2.3.4/ch1/live.m3u8
	Source string
	//sync_output=/dev/shm/videos
	Sync_output string
	//sync_rewrite=
	Sync_rewrite string
	//record_output=/data/videos
	Record_output string
	//record_rewrite=
	Record_rewrite string
	//use_localtime=false
	Use_localtime bool
	//use_segment_time=true
	Use_segment_time bool
	//segment_timestamp=
	Segment_timestamp string
	//segment_duration=5
	//Segment_duration int
	//redirect=
	Redirect string
	//timeout=
	Timeout int
	//user_agent=
	User_agent string
}


type Rewriter interface {
	PlaylistRewrite(s string) string
	SegmentRewrite(s string) string
}

type SyncRewriter struct {

}

func (self *SyncRewriter) PlaylistRewrite(s string) string {
	return ""
}

func (self *SyncRewriter) SegmentRewrite(s string) string {
	return ""
}

type RecordRewriter struct {

}

func (self *RecordRewriter) PlaylistRewrite(s string) string {
	return ""
}

func (self *RecordRewriter) SegmentRewrite(s string) string {
	return ""
}

func NewSyncRewriter(cmd string) (*SyncRewriter, error) {
	sr := new(SyncRewriter)
	return sr, nil
}

func NewRecordRewriter(cmd string) (*RecordRewriter, error) {
	rr := new(RecordRewriter)
	return rr, nil
}


type SyncOption struct {
	Option
	sync_rewriter   Rewriter
	record_rewriter Rewriter
}

type LinkOptions struct {
	Live_Channels []*Option
}


func Load_HLS_Links(filename string) (links []*SyncOption) {
	linkoptions := LinkOptions{}
	if _, e := toml.DecodeFile(filename, &linkoptions); nil != e {
		return
	}else{
		for _, op := range linkoptions.Live_Channels {
			so, e := Build_Sync_Option(op)
			if nil != e {
				log.Errorln("Failed to build sync option:",e, "Line:", op)
				continue
			}
			links = append(links, so)
		}
	}
	return links
}

func AssignOption(dest *Option, src *Option, with_source bool){
	dest.Segment_timestamp = src.Segment_timestamp
	//dest.Segment_duration = src.Segment_duration
	dest.Use_localtime = src.Use_localtime
	dest.Record_output = src.Record_output
	dest.Record_rewrite = src.Record_rewrite
	dest.Redirect = src.Redirect
	dest.Sync_output = src.Sync_output
	dest.Sync_rewrite = src.Sync_rewrite
	dest.Timeout = src.Timeout
	dest.Use_segment_time = src.Use_segment_time
	dest.User_agent = src.User_agent
	if with_source {
		dest.Source = src.Source
	}
}


func Build_Sync_Option(option *Option, default_options ...*Option) (so *SyncOption, err error) {
	so = new(SyncOption)
	AssignOption(&so.Option, option, true)
	for _, op := range default_options {
		AssignOption(&so.Option, op, false)
	}
	so.sync_rewriter, err = NewSyncRewriter(so.Sync_rewrite)
	if nil != err {
		return nil, err
	}
	so.record_rewriter, err = NewRecordRewriter(so.Record_rewrite)
	if nil != err {
		return nil, err
	}
	return so, nil
}