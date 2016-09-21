package main

import (
	log "github.com/Sirupsen/logrus"
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
	SFX_RUNNING = "_RUNNING"
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
	ret := []string{}
	for i:=0; i< limit; i++ {
		l, e := self._pdb.LPop(self.key)
		if nil == e {
			ret = append(ret, l)
			self._pdb.HSet(self.key+SFX_RUNNING, l, 1)
		}else{
			break
		}
	}
	return ret, nil
}

func (self *Dl_Redis) SubmitResult(link string, dest string, ret_code int, ret_msg string) {
	log.Infoln("DL >", link, dest, ret_code, ret_msg)
	ret := map[string]interface{}{"link":link, "dest": dest, "ret_code": ret_code, "ret_msg": ret_msg}
	if ret_code != 0 {
		self._pdb.HSet(self.key+SFX_FAILED, link, ret)
	}else {
		self._pdb.HSet(self.key+SFX_SUCCESS, link, ret)
	}
	self._pdb.HDel(self.key+SFX_RUNNING, link)
}
