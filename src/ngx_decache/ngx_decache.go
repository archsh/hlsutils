package main

import (
	"flag"
	"fmt"
	"fxhelpers/ngx_md5"
	"os"
)

const VERSION = "0.0.1"

func usage() {
	os.Stderr.Write([]byte(fmt.Sprintf("\nngx_decache %v - A Nginx Cached files processing tool.\n", VERSION)))
	os.Stderr.Write([]byte("Copyright (C) 2015 Mingcai SHEN. Licensed for use under the GNU GPL version 3.\n\n"))
}

func process_cache(uri string, cacheRoot string, scanSegments bool, detectOnly bool) {
	fmt.Printf("process_cache: \n\turi=%s \n\tcacheRoot=%s \n\t scanSegments=%v  detectOnly=%v\n",
		uri, cacheRoot, scanSegments, detectOnly)
}

func main() {
	var cacheRoot string
	var uriList string
	flag.StringVar(&cacheRoot, "p", "", "Cache root path.")
	flag.StringVar(&uriList, "f", "", "URI list file. A text file stores multiple URIs and seperated in lines.")
	scanSegments := flag.Bool("s", false, "Scan M3U8 segments from cached URIs.")
	detectOnly := flag.Bool("d", false, "Detect cache only, do not remove cached files.")
	flag.Parse()
	fmt.Printf("Testing: %s \n", ngx_md5.Md5sum("testing"))

	if flag.NArg() < 1 && uriList == "" {
		usage()
		flag.PrintDefaults()
		os.Stderr.Write([]byte("\n"))
		os.Exit(2)
	}
	if uriList != "" {
		// Processing URIs from file
	}
	if flag.NArg() > 0 {
		// Processing URIs from arguments
		for i := 0; i < flag.NArg(); i++ {
			process_cache(flag.Arg(i), cacheRoot, *scanSegments, *detectOnly)
		}
	}

}
