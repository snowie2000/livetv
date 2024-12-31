package main

import (
	"context"
	"crypto/tls"
	"flag"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	_ "github.com/joho/godotenv/autoload"
	"github.com/natefinch/lumberjack"
	"github.com/robfig/cron/v3"
	"github.com/snowie2000/livetv/global"
	"github.com/snowie2000/livetv/route"
	"github.com/snowie2000/livetv/service"
)

func main() {
	pwd := flag.String("pwd", "", "reset password")
	listen := flag.String("listen", ":9000", "listening address")
	disableProtection := flag.Bool("disable-protection", false, "temporarily disable token protection")
	flag.Parse()
	datadir := os.Getenv("LIVETV_DATADIR")
	if datadir == "" {
		ex, err := os.Executable()
		if err != nil {
			panic(err)
		}
		datadir = filepath.Join(filepath.Dir(ex), "data")
		os.Setenv("LIVETV_DATADIR", datadir)
	}
	os.Mkdir(datadir, os.ModePerm)

	if *pwd != "" {
		// reset password
		err := global.InitDB(datadir + "/livetv.db")
		if err != nil {
			log.Panicf("init: %s\n", err)
		}
		err = global.SetConfig("password", *pwd)
		if err == nil {
			log.Println("Password has been changed.")
		} else {
			log.Println("Failed to reset password:", err.Error())
		}
		return
	}

	if *disableProtection {
		os.Setenv("LIVETV_FREEACCESS", "1")
	}

	binding := os.Getenv("LIVETV_LISTEN")
	if binding == "" {
		binding = *listen
		os.Setenv("LIVETV_LISTEN", binding)
	}
	rand.Seed(time.Now().UnixNano())
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("Server listen", binding)
	log.Println("Server datadir", datadir)

	log.SetOutput(io.MultiWriter(os.Stderr, &lumberjack.Logger{
		Filename:   datadir + "/livetv.log",
		MaxSize:    10, // megabytes
		MaxBackups: 3,
		MaxAge:     1,    //days
		Compress:   true, // disabled by default
	}))
	err := global.InitDB(datadir + "/livetv.db")
	if err != nil {
		log.Panicf("init: %s\n", err)
	}
	log.Println("LiveTV starting...")
	go service.LoadChannelCache()
	c := cron.New()
	//_, err = c.AddFunc("0 */3 * * *", service.UpdateURLCache)
	_, err = c.AddFunc("@every 3h", service.UpdateURLCache)
	if err != nil {
		log.Panicf("preloadCron: %s\n", err)
	}
	c.Start()
	sessionSecert, err := global.GetConfig("password")
	if err != nil {
		sessionSecert = "sessionSecert"
	}
	// ignore tls cert error
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()
	store := cookie.NewStore([]byte(sessionSecert))
	/* CORS */
	/*config := cors.DefaultConfig()
	config.AllowOrigins = []string{"http://localhost:8000"}
	config.AllowCredentials = true
	router.Use(cors.New(config))*/
	router.Use(sessions.Sessions("mysession", store))
	// router.Static("/", "./web")
	route.Register(router)
	srv := &http.Server{
		Addr:    binding,
		Handler: router,
	}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Panicf("listen: %s\n", err)
		}
	}()
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shuting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Panicf("Server forced to shutdown: %s\n", err)
	}
	log.Println("Server exiting")
}
