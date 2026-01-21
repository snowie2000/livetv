# LiveTV 
[中文](doc/Howto_cn.md)

Aggregate IPTV feeds in one station and enjoy!

## Install 

Download the latest binary from [releases](https://github.com/snowie2000/livetv/releases) 

## Docker

Precompiled images are available as `flyingsnow2000/livetv:latest` and `flyingsnow2000/livetv:version_number`. Only linux/amd64 is currently available. Images for other platforms require a custom image build.

`Python 3.13`, `yt-dlp latest` and `bgutil-ytdlp-pot-provider` are included in the image and can be used in the yt-dlp extractorargs directly.


### How to start
```
docker run --name livetv -p 9000:9000 -v /path/to/data:/opt/livetv/data -d flyingsnow2000/livetv:latest
```

## Usage

```
Usage of ./livetv:
  -disable-protection
        temporarily disable token protection
  -listen string
        listening address (default ":9000")
  -pwd string
        reset password
```

Default password is "password".

First you need to know how to access your host from the outside, if you are using a VPS or a dedicated server, you can visit `http://your_ip:9500` and you should see the following screen.

![index_page](pic/index-en.png)

First of all, you need to click the gear button and "Auto Fill" in the setting area, set the correct URL, then click "Ok".

Then you can add a channel. After the channel is added successfully, you can play the M3U8 file from the address column.

When you use Kodi or similar player, you can consider using the M3U file URL in the playlist field, and a playlist containing all the channel information will be generated automatically.

To protect your service from unauthorized access, you can set a secret in the settings dialog and then all your playlist and proxy services will need a unique token to access (based on your secret).
