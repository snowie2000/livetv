
# 解析器
选择正确的解析器可以帮助您访问之前无法播放的源，并减少对php等解析脚本的调用，避免频繁调用解析API导致ip被封禁，或者速度缓慢的问题。

您可以使用linux或Windows10以后系统自带的curl来判断您源的类型。

以下会对目前的每个解析器做相应的描述并帮助您选择正确的解析器。


### httpRedirect
用途：
- 该解析器可以识别http跳转，解析出真实的m3u8地址
- 如果您在php脚本中看到了类似`header('Location: https://某个网址/任何内容');`的代码，您应该选择这个解析器
- 该解析器可以解析主播放列表并自动选择其中质量最高的源

判断方法：
请执行以下命令
```bash
curl http://example.com/live.php?channel=xxx -vv 2>&1 | grep ocation
```
如果以上命令返回了类似这样的内容：
>  File STDIN:
  < Location: http://（或https://）某个网址/任何内容

则您应该选择`httpRedirect`解析器

### rtmp
用途：
- 该解析器可以解析rtmp直播地址
- 该解析器可以解析http跳转到rtmp的地址
- 该解析器可以将rtmp协议转换为flv协议，以便tvbox等软件播放
- **使用该解析器将通过livetv代理流，因此如果在云服务器上部署，请注意流量使用！**

判断方法：
- 在httpRedirect一节的命令中，如果返回值不是http或https开头，而是rtmp开头则您应该选择`rtmp`解析器
- 如果您的视频地址本来就是rtmp协议的，则您应该选择`rtmp`解析器

### direct
用途：
- 该解析器接受一个m3u8地址，并补全地址后转发
- 如果您的php脚本直接返回了m3u8地址，您应该选择这个解析器
- 该解析器可以解析主播放列表并自动选择其中质量最高的源

判断方法：
请执行
```bash
curl http://example.com/live.php?channel=xxx -vv 2>&1 | grep M3U
```
如果以上命令返回了任意内容，则您应该选择`direct`解析器

### repeater
用途：
- 该解析器接受一个m3u8地址，并直接转发不做任何修改
- 如果您使用的是一个静态源，只是想在livetv中统一管理，您应该选择这个解析器

判断方法:
如果您的源是类似
`http://example.com/xxx.m3u8`这样的地址，您应该选择`repeater`解析器


### youtube
用途：
- 该解析器可以解析youtube直播地址
- 该解析器会直接选择youtube直播中质量最高的源
- 支持任意格式的youtube直播地址，移动端pc端均可

如果您使用youtube直播作为您iptv的源，请选择此解析器

### yt-dlp
用途：
- 该解析器可以解析youtube直播地址
- 该解析器会直接选择youtube直播中质量最高的源
- 支持任意格式的youtube直播地址，移动端pc端均可
- 该解析器使用yt-dlp来解析youtube直播地址，可以解析更多的youtube直播地址
- 使用本解析器，您需要提前下载yt-dlp程序，并将其放在livetv程序的同一目录下，否则将解析失败
- 使用yt-dlp解析器将调用第三方程序，因此速度较慢，并会占用更多系统资源，但可能解析一些内建youtube解析器不能正常处理的情况。

----

下一章：[流代理](TSProxy_cn.md)