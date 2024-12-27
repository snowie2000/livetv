# 开发

## 解释器指令

内置的http解释器支持从源的返回值中读取规定的部分json指令，从而实现模拟头部通过认证等功能。

以下为解释器支持的完整json格式：
```json
{
  "logo": "https://example.com/logo.png", // 可选频道logo
  "headers": {
    "header1": "value1",
    "header2": "value2"
    ... // 可选，自定义头部
  }
}
```

logo将在m3u中作为频道图标输出。

headers将在获取m3u8时自动添加到请求中，如果代理了流，则代理时也会使用这些头部。

以下是一个能支持该功能的php直播流解释器示例：
```php
<?php
// 主体跳转到真实直播流地址
header("Location: https://example.com/live.m3u8");
// 返回的内容是一个json
header('Content-Type: application/json');
// 直播地址需要header认证，所以我们指示livetv添加header
echo json_encode([
  "logo" => "https://example.com/logo.png",
  "headers" => [
    "User-Agent" => "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/58.0.3029.110 Safari/537.3",
    "Referer" => "https://example.com"
  ]
]);
?>
```

## 开发解析器

欢迎开发者为livetv开发新的解析器。

解析器位于`plugin`文件夹下，每个解析器相互独立，也可互相嵌套。

解析器目前支持以下接口，您可以根据需要实现：

```go
// 该接口必须实现，输入直播源地址和代理信息，返回解析后的直播信息
// previousExtraInfo 包含了上一次解析时记录的额外信息
type Plugin interface {
	Parse(liveUrl string, proxyUrl string, previousExtraInfo string) (info *model.LiveInfo, error error)
}

// 可选
// 在请求m3u8实际地址前回调，可以对请求进行修改
type Transformer interface {
	Transform(req *http.Request, info *model.LiveInfo) error
}

// 可选
// 对接收到的m3u8内容进行健康检查，返回错误将触发重新解析（有重试限制）
type HealthCheck interface {
	Check(content string, info *model.LiveInfo) error
}

// 可选
// 给予频道的具体信息，直接处理频道的数据，如果不返回错误，则外部将不再按标准m3u8流程继续处理
// 可用于serve非m3u8的直播源，如rtmp, rtsp等
type FeedHost interface {
	Host(c *gin.Context, info *model.LiveInfo) error
}

// 可选
// 解析器可提供子节目列表（虚拟节目单），如果提供了该接口，将会在节目单中显示子节目
// 同时该节目单本身会在列表中隐藏
type ChannalProvider interface {
	Channels(parentChannel *model.Channel, liveInfo *model.LiveInfo) []*model.Channel
}

// 可选
// 对最终ts链接进行转换，可用于添加头部，自定义代理等
type TsTransformer interface {
	TransformTs(rawLink string, tsLink string, info *model.LiveInfo) string
}

// 可选
// 允许解析器自由构建m3u8内容而不通过默认的从互联网获取m3u8
type Forger interface {
	ForgeM3U8(info *model.LiveInfo) (baseUrl string, body string, err error)
}
```