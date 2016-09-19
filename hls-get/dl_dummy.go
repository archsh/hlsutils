package main

import (
	log "github.com/Sirupsen/logrus"
	"errors"
)

type DL_Interface interface {
	NextLinks(limit int) ([]string, error)
	SubmitResult(link string, dest string, ret_code int, ret_msg string)
}


type Dl_Dummy struct {
	links []string
	cursor int
}

func NewDummyDl(links []string) *Dl_Dummy {
	dl := new(Dl_Dummy)
	dl.links = links
	return dl
}

func (self *Dl_Dummy) NextLinks(limit int) ([]string, error) {
	if self.cursor >= len(self.links){
		return nil, errors.New("Out of index.")
	}else{
		ret := self.links[self.cursor:self.cursor+limit]
		self.cursor += limit
		return ret, nil
	}
}

func (self *Dl_Dummy) SubmitResult(link string, dest string, ret_code int, ret_msg string) {
	log.Infoln("DL >", link, dest, ret_code, ret_msg)
}
