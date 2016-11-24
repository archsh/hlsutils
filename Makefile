.PHONY: hls-get hls-sync live-copy ngx-decache

all: hls-get hls-sync live-copy ngx-decache

hls-get hls-sync live-copy ngx-decache:
	@echo "Installing $@ ..."
	@go install github.com/archsh/hlsutils/$@

