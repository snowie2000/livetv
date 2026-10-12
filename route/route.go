package route

import (
	"github.com/gin-gonic/gin"
	"github.com/snowie2000/livetv/handler"
)

func Register(r *gin.Engine) {
	r.OPTIONS("/", handler.CORSHandler)
	r.GET("/lives.m3u", handler.M3UHandler)
	r.GET("/proxy", handler.ReverseProxyHandler)
	r.GET("/lives.txt", handler.TXTHandler)
	r.GET("/live.m3u8", handler.LiveHandler)
	r.HEAD("/live.m3u8", handler.LivePreHandler)
	r.GET("/live.ts", handler.TsProxyHandler)
	r.GET("/playlist.m3u8", handler.M3U8ProxyHandler)
	r.GET("/cache.txt", handler.CacheHandler)

	// Public APIs
	r.GET("/api/crsf", handler.CRSFHandler)
	r.POST("/api/login", handler.LoginActionHandler)
	r.GET("/api/captcha", handler.CaptchaHandler)
	r.Any("/fetch", handler.FetchHandler)

	// Protected Admin APIs
	admin := r.Group("/")
	admin.Use(handler.AuthRequired())
	{
		admin.GET("/api/channels", handler.ChannelListHandler)
		admin.GET("/api/plugins", handler.PluginListHandler)
		admin.POST("/api/newchannel", handler.NewChannelHandler)
		admin.POST("/api/updatechannel", handler.UpdateChannelHandler)
		admin.GET("/api/getconfig", handler.GetConfigHandler)
		admin.GET("/api/delchannel", handler.DeleteChannelHandler)
		admin.POST("/api/updconfig", handler.UpdateConfigHandler)
		admin.GET("/api/auth", handler.AuthProbeHandler)
		admin.GET("/api/category", handler.CategoryHandler)
		admin.GET("/log", handler.LogHandler)
		admin.GET("/api/logout", handler.LogoutHandler)
		admin.POST("/api/changepwd", handler.ChangePasswordHandler)
	}

	// Static Web UI routes
	r.GET("/", handler.IndexHandler)
	r.GET("/:path", handler.IndexHandler)
}
