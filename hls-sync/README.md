# hls-sync

Sync hls live stream to local disk.

## Scenarios:

   * Sync live hls streams from remote hls server.
   * Record live streams to local disks.

## Usage:

    hls-sync [OPTIONS,...] SOURCE_URL1 ...

You can run with several URL as failover mechanism. `hls-sync` the first one and than the second one if there's a failure.

### Options:

- `c` string
    Configuration file instead of command line parameters. Default empty means using parameters.
    See "Configuration Example" for detail.
- `O` string
    Output path. A base path for storage segments and play list.
- `SR` string
    Segment filename rewrite rule. Default empty means simply copy.
    For example: "%Y/%m/%d/%H/live-%04d.ts"
- `TZ` int
    Timezone shift. Default 0.
- `TD` int
    Target duration in seconds. Default 0.
- `L` string
    Logging output file. Default 'stdout'.
- `LV` string
    Logging level. Default 'INFO'.
- `SP` bool
    Save play list file. Default false.
- `TO` int
    Request timeout. Default 5.
- `R` int
    Retries. Default 1.
- `UA` string
    User Agent. Default 'hls-sync v${VERSION}'.
- `UL` bool
    Use local timestamp.
- `US` bool
    Use segment timestamp.
- `ST` string
    Segment timestamp format.
- `RM` bool
    Remove old segments.

### Example

### Record stream:
    ./hls-sync -O /tmp/channel1 -SR '%Y/%m/%d/%H/live-%04d.ts' -RS -RD 10 -LV DEBUG -UL http://live1.example.com/chan01/live.m3u8 http://live2.example.com/chan01/live.m3u8

### Sync stream only:
    ./hls-sync -O /tmp/channel1 -SP -RM -LV DEBUG http://live1.example.com/chan01/live.m3u8 http://live2.example.com/chan01/live.m3u8    
    
## Configuration Example

`hls-sync` use a TOML configuration file. Please check the following example:

