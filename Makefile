.PHONY: hls-get hls-sync live-copy ngx-decache

all: hls-get hls-sync live-copy ngx-decache


hls-get hls-sync live-copy ngx-decache:
	@echo "Building $@ ..."
	@go build github.com/archsh/hlsutils/$@

