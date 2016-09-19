package main

import (
	log "github.com/Sirupsen/logrus"
	"errors"
	"github.com/gosexy/redis"
	"strconv"
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



func redis_connect(host string, port int, db int) (client *redis.Client, e error) {
	client = redis.New()

	e = client.Connect(host, uint(port))
	if e != nil {
		return
	}
	client.Select(int64(db))
	return
}

func redis_get_indicator(c *redis.Client, k string) (result bool) {
	if c == nil {
		return false
	}
	r, e := c.Get(k + "_indicator")
	// log.Printf("%s:> %s", k+"_indicator", r)
	if e != nil {
		return false
	}
	i, e := strconv.Atoi(r)
	// log.Printf("i=%d, e=%v", i, e)
	if e == nil && i > 0 {
		return true
	}
	return false
}

func redis_get_link(c *redis.Client, k string) (link *string) {
	if c == nil {
		//err = error("Client can not be nil.")
		return nil
	}
	l, _ := c.LPop(k)
	// c.LRange(key, start, stop)
	log.Printf("Get link: %s", l)
	link = &l
	return
}

func redis_set_finished(c *redis.Client, k string, link *string) (err error) {
	err = nil
	if c == nil {
		return
	}
	c.LPush(k+"_finished", *link)
	return
}

func redis_set_failed(c *redis.Client, k string, link *string) (err error) {
	err = nil
	if c == nil {
		return
	}
	c.LPush(k+"failed", *link)
	return
}
