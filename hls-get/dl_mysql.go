package main

import (
	log "github.com/Sirupsen/logrus"
	"errors"
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	"fmt"
)

/*
The following Table Structure is for hls-get to download from a MySQL db table.
`url` field is the source url for downloading,
`dest` field will be filled with file saved path after downloaded,
`ret_code` and `ret_msg` indicates the download result, 0 and empty message means DONE well.
*/

/*
-- ----------------------------
-- Table structure for download_list
-- ----------------------------
DROP TABLE IF EXISTS `hlsget_downloads`;
CREATE TABLE `hlsget_downloads` (
`id` int(11) NOT NULL AUTO_INCREMENT,
`url` varchar(256) NOT NULL DEFAULT '',
`dest` varchar(256) DEFAULT NULL,
`ret_code` int(11) DEFAULT '0',
`ret_msg` varchar(128) DEFAULT NULL,
PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
*/

/****
 The following script shows load download list from epgdb_vod.pulish_movie:

 INSERT INTO hlsgetdb.hlsget_downloads (url, ret_code) SELECT `guid`, -1 FROM epgdb_vod.publish_movie;
 */

type Dl_MySQL struct {
	host string
	port uint
	db string
	table string
	username string
	password string
	_pdb *sql.DB
}

func NewMySQLDl(host string, port uint, db string, table string, username string, password string) *Dl_MySQL {
	dl := new(Dl_MySQL)
	dl.host = host
	dl.port = port
	dl.username = username
	dl.password = password
	dl.db = db
	dl.table = table
	var dburi string
	if password != "" {
		dburi = fmt.Sprintf("%s:%s@%s:%d/%s", username, password, host, port, db)
	}else{
		dburi = fmt.Sprintf("%s@%s:%d/%s", username, host, port, db)
	}

	pdb, err := sql.Open("mysql", dburi)
	if err != nil {
		return nil
	}else{
		dl._pdb = pdb
	}
	return dl
}

func (self *Dl_MySQL) NextLinks(limit int) ([]string, error) {
	return nil, errors.New("Not implemented!")
}

func (self *Dl_MySQL) SubmitResult(link string, dest string, ret_code int, ret_msg string) {
	log.Infoln("DL >", link, dest, ret_code, ret_msg)
}
