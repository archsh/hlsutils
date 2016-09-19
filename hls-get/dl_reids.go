package main

import (
	log "github.com/Sirupsen/logrus"
	"errors"
	"github.com/gosexy/redis"
)

type Dl_Redis struct {
	host string
	port uint
	db int
	key string
	_pdb *redis.Client
}

const (
	SFX_SUCCESS = "_SUCCESS"
	SFX_FAILED  = "_FAILED"
)

func NewRedisDl(host string, port uint, password string, db int, key string) *Dl_Redis {
	dl := new(Dl_Redis)
	dl.host = host
	dl.port = port
	dl.db = db
	dl.key = key
	dl._pdb = redis.New()
	err := dl._pdb.Connect(host, port)
	if nil != err {
		return nil
	}
	if password != "" {
		dl._pdb.Auth(password)
	}
	if db > 1 {
		dl._pdb.Select(int64(db))
	}
	return dl
}

func (self *Dl_Redis) NextLinks(limit int) ([]string, error) {
	return nil, errors.New("Not implemented!")
}

func (self *Dl_Redis) SubmitResult(link string, dest string, ret_code int, ret_msg string) {
	log.Infoln("DL >", link, dest, ret_code, ret_msg)
}
