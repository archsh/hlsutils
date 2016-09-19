package main

import (
	//"bytes"
	//"flag"
	//"fmt"
	//"github.com/golang/groupcache/lru"
	//"github.com/gosexy/redis"
	//"github.com/kz26/m3u8"
	//"io"
	//"io/ioutil"
	//log "github.com/Sirupsen/logrus"
	//"net/http"
	//"net/url"
	//"os"
	//"path"
	//"regexp"
	//"strconv"
	//"strings"
	//"time"
	//"github.com/archsh/hlsutils/helpers/logging"
	"github.com/rwtodd/sed-go"
	"strings"
	"os"
)


type StringRewriter interface {
	RunString(string) string
}


type PathRewriter struct {
	engine *sed.Engine
}

func (self *PathRewriter) RunString(input string) string {
	if nil != self.engine {
		s, e := self.engine.RunString(input)
		if nil != e {
			return input
		}else{
			return s
		}
	}else{
		return input
	}
}

func NewPathRewriter(cmd string) (pr *PathRewriter) {
	engine, err := sed.New(strings.NewReader(cmd))
	pr = new(PathRewriter)
	if nil == err {
		pr.engine = engine
	}
	return
}

type SegmentRewriter struct {

}

func (self *SegmentRewriter) RunString(input string) string {
	return  input
}

func NewSegmentRewriter(cmd string) (sr *SegmentRewriter){
	sr = new(SegmentRewriter)
	return
}

func exists(path string) (b bool) {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	return false
}