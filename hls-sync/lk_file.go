package main

type SyncOption struct {
	source string
	sync_rewrite string
	record_rewrite string
}


func Load_HLS_Links(filename string) (links []*SyncOption) {

	return links
}


func Build_Sync_Option(link string, sync_rewrite string, record_rewrite string) *SyncOption {
	so := new(SyncOption)
	so.source = link
	return so
}