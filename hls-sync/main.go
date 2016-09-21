package main

import (
	log "github.com/Sirupsen/logrus"
	"os"
	"os/signal"
	"fmt"
	"strings"
	"flag"
	"time"
	"bufio"
)

func main() {
	duration := flag.Duration("t", time.Duration(0), "Recording duration (0 == infinite)")
	useLocalTime := flag.Bool("l", false, "Use local time to track duration instead of supplied metadata")
	deleteOld := flag.Bool("d", false, "Delete old segments.")
	retryTimes := flag.Int("r", 0, "Retry if failed. 0: Never retry, <0: Keep retry, >0: Retry times.")
	var outputDir string
	flag.StringVar(&outputDir, "o", ".", "Output path. default is '.'")
	var listFilename string
	flag.StringVar(&listFilename, "u", "", "URL list file for multiple sync.")
	flag.StringVar(&USER_AGENT, "ua", fmt.Sprintf("hls-sync/%v", VERSION), "User-Agent for HTTP Client")
	flag.Parse()

	endChan := make(chan bool)

	os.Stderr.Write([]byte(fmt.Sprintf("hls-sync %v - A Realtime HTTP Live Streaming (HLS) Synchronizar.\n", VERSION)))
	os.Stderr.Write([]byte("Copyright (C) 2015 Mingcai SHEN <archsh#gmail.com>. Licensed for use under the GNU GPL version 3.\n"))

	if flag.NArg() < 1 && listFilename == "" {
		os.Stderr.Write([]byte("Usage: hls-sync [Options] media-playlist-url\n"))
		os.Stderr.Write([]byte("Options:\n"))
		flag.PrintDefaults()
		os.Stderr.Write([]byte("\n"))
		os.Exit(2)
	}

	var linkList []string

	if listFilename != "" {
		f, e := os.Open(listFilename)
		if e != nil {
			fmt.Printf("Open file %s failed! [%v]\n", listFilename, e)
			os.Exit(2)
		}
		defer f.Close()
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			linkList = append(linkList, scanner.Text())
		}
	} else {
		linkList = flag.Args()
	}

	stsChan := make(chan *Status, 64)
	stsMap := make(map[string]*Status, 1024)

	for _, link := range linkList {
		log.Printf("Start to get link: %s \n", link)
		if !strings.HasPrefix(link, "http") {
			fmt.Println("Media playlist url must begin with http/https")
			os.Exit(2)
		}
		msChan := make(chan *Download, 1024)
		go getPlaylist(link, outputDir, *duration, *deleteOld, *useLocalTime, *retryTimes, msChan)
		go downloadSegment(msChan, *duration)
	}

	go update_status(stsChan, stsMap)

	term_c := make(chan os.Signal, 1)

	signal.Notify(term_c, os.Interrupt)
	for {
		select {
		case <-term_c:
			log.Printf("User controled terminated.\n")
			os.Exit(0)
		case <-endChan:
			log.Printf("Sync finished!\n")
			break
		}
	}
}
