package route

import (
	"github.com/gin-gonic/gin"
	"github.com/snowie2000/livetv/handler"
)

func Register(r *gin.Engine) {
	r.OPTIONS("/", handler.CORSHandler)
	r.GET("/lives.m3u", handler.M3UHandler)
	r.GET("/lives.txt", handler.TXTHandler)
	r.GET("/live.m3u8", handler.LiveHandler)
	r.HEAD("/live.m3u8", handler.LivePreHandler)
	r.GET("/live.ts", handler.TsProxyHandler)
	r.GET("/playlist.m3u8", handler.M3U8ProxyHandler)
	r.GET("/cache.txt", handler.CacheHandler)

	r.GET("/api/channels", handler.ChannelListHandler)
	r.GET("/api/plugins", handler.PluginListHandler)
	r.GET("/api/crsf", handler.CRSFHandler)
	r.POST("/api/newchannel", handler.NewChannelHandler)
	r.POST("/api/updatechannel", handler.UpdateChannelHandler)
	r.GET("/api/getconfig", handler.GetConfigHandler)
	r.GET("/api/delchannel", handler.DeleteChannelHandler)
	r.POST("/api/updconfig", handler.UpdateConfigHandler)
	r.GET("/api/auth", handler.AuthProbeHandler)
	r.GET("/api/category", handler.CategoryHandler)
	r.GET("/log", handler.LogHandler)
	// r.GET("/login", handler.LoginViewHandler)
	r.POST("/api/login", handler.LoginActionHandler)
	r.GET("/api/logout", handler.LogoutHandler)
	r.POST("/api/changepwd", handler.ChangePasswordHandler)
	r.GET("/api/captcha", handler.CaptchaHandler)
	r.GET("/", handler.IndexHandler)
	r.GET("/fetch", handler.FetchHandler)
	r.GET("/:path", handler.IndexHandler)
}
