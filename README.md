# HLS Utils

## hls-get

### Scenarios:

   * Simple mode: download one or multiple URL without DB support.
   * Redis support: download multiple URL via REDIS LIST.
   * MySQL support: download multiple URL via MySQL DB Table.

### Usage:

    hls-get [OPTIONS,...] [URL1,URL2,...]

### Options:

- C int
      Concurrent tasks. (default 5)
- L string
      Logging output file. Default 'stdout'.
- M string
      Source mode: redis, mysql. Empty means source via command args.
- MD string
      MySQL database. (default "hlsgetdb")
- MH string
      MySQL host. (default "localhost")
- MN string
      MySQL username. (default "root")
- MP int
      MySQL port. (default 3306)
- MT string
      MySQL table. (default "hlsget_downloads")
- MW string
      MySQL password.
- O string
      Output directory. (default ".")
- PR string
      Rewrite output path method. Empty means simple copy. 
      The implement of Path Rewrite is powered by 'https://github.com/rwtodd/sed-go', please check the document when using it.
      Normally it's like a sed command, but with differences:
          
| Go-sed          |  Traditional RE   | Notes                             |
| --------------- | ----------------- | --------------------------------- |
|  s/a(bc*)d/$1/g |  s/a\\(bc*\\)d/\1/g | Don't escape (); Use $1, $2, etc. |
|  s/(?s).//      |  s/.//            | If you want dot to match \n, use (?s) flag.  |

- R int
      Retry times if download fails.
- RD int
      Redis db num.
- RH string
      Redis host. (default "localhost")
- RK string
      List key name in redis. (default "HLSGET_DOWNLOADS")
- RP int
      Redis port. (default 6379)
- RR string
      Redirect server request.
- RW string
      Redis password.
- S  bool
      Skip if exists.
- SR string
      Rewrite segment name method. Empty means simple copy.
- TO int
      Request timeout in seconds. (default 20)
- TT int
      Total download links.
- UA string
      UserAgent. (default "hls-get v0.9.4")

### Data Structure of MySQL:

  The following Table Structure is for hls-get to download from a MySQL db table.
  
  `url` field is the source url for downloading,
  
  `dest` field will be filled with file saved path after downloaded,
  
  `status` 0 means for download, 1 means downloading, 2 = success, less than 0 means failed.
  
  `ret_code` and `ret_msg` indicates the download result, 0 and empty message means DONE well.
 
  -- Table structure for hlsget_downloads
  
      DROP TABLE IF EXISTS `hlsget_downloads`;
      CREATE TABLE `hlsget_downloads` (
        `id` int(11) NOT NULL AUTO_INCREMENT,
        `url` varchar(256) NOT NULL,
        `status` int(11) NOT NULL DEFAULT '0',
        `dest` varchar(256) DEFAULT NULL,
        `ret_code` int(11) DEFAULT '0',
        `ret_msg` varchar(128) DEFAULT NULL,
        PRIMARY KEY (`id`),
        UNIQUE KEY `url` (`url`)
      ) ENGINE=InnoDB AUTO_INCREMENT=393211 DEFAULT CHARSET=latin1;
 
   The following script shows load download list from epgdb_vod.pulish_movie:
  
    INSERT INTO hlsgetdb.hlsget_downloads (url, ret_code) SELECT `guid`, 0 FROM epgdb_vod.publish_movie WHERE `guid` <> "";

### Data Structure of Redis:

  Simply push your download list to Redis as a Redis List.


## hls-sync

