package main

import (
	//log "github.com/Sirupsen/logrus"
	//"github.com/BurntSushi/toml"
)

type Rewriter interface {
	PlaylistRewrite(s string) string
	SegmentRewrite(s string) string
}

type SyncRewriter struct {

}
