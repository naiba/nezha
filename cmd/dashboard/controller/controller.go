package controller

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	jwt "github.com/appleboy/gin-jwt/v2"
	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	docs "github.com/naiba/nezha/cmd/dashboard/docs"
	"github.com/naiba/nezha/model"
	"github.com/naiba/nezha/service/singleton"
)

func ServeWeb() http.Handler {

	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	if singleton.Conf.Debug {
		gin.SetMode(gin.DebugMode)
		pprof.Register(r)
	}
	if singleton.Conf.Debug {
		log.Printf("NEZHA>> Swagger(%s) UI available at http://localhost:%d/swagger/index.html", docs.SwaggerInfo.Version, singleton.Conf.ListenPort)
		r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerfiles.Handler))
	}

	r.Use(recordPath)
	routers(r)

	return r
}

func routers(r *gin.Engine) {
	authMiddleware, err := jwt.New(initParams())
	if err != nil {
		log.Fatal("JWT Error:" + err.Error())
	}
	if err := authMiddleware.MiddlewareInit(); err != nil {
		log.Fatal("authMiddleware.MiddlewareInit Error:" + err.Error())
	}
	api := r.Group("api/v1")
	api.POST("/login", authMiddleware.LoginHandler)

	optionalAuth := api.Group("", optionalAuthMiddleware(authMiddleware))
	optionalAuth.GET("/ws/server", commonHandler(serverStream))
	optionalAuth.GET("/server-group", commonHandler(listServerGroup))

	auth := api.Group("", authMiddleware.MiddlewareFunc())

	auth.GET("/refresh_token", authMiddleware.RefreshHandler)

	auth.POST("/terminal", commonHandler(createTerminal))
	auth.GET("/ws/terminal/:id", commonHandler(terminalStream))

	auth.GET("/file", commonHandler(createFM))
	auth.GET("/ws/file/:id", commonHandler(fmStream))

	auth.GET("/user", commonHandler(listUser))
	auth.POST("/user", commonHandler(createUser))
	auth.POST("/batch-delete/user", commonHandler(batchDeleteUser))

	auth.GET("/service", commonHandler(listService))
	auth.POST("/service", commonHandler(createService))
	auth.PATCH("/service/:id", commonHandler(updateService))
	auth.POST("/batch-delete/service", commonHandler(batchDeleteService))

	auth.POST("/server-group", commonHandler(createServerGroup))
	auth.PATCH("/server-group/:id", commonHandler(updateServerGroup))
	auth.POST("/batch-delete/server-group", commonHandler(batchDeleteServerGroup))

	auth.GET("/notification-group", commonHandler(listNotificationGroup))
	auth.POST("/notification-group", commonHandler(createNotificationGroup))
	auth.PATCH("/notification-group/:id", commonHandler(updateNotificationGroup))
	auth.POST("/batch-delete/notification-group", commonHandler(batchDeleteNotificationGroup))

	auth.GET("/server", commonHandler(listServer))
	auth.PATCH("/server/:id", commonHandler(updateServer))
	auth.POST("/batch-delete/server", commonHandler(batchDeleteServer))

	auth.GET("/notification", commonHandler(listNotification))
	auth.POST("/notification", commonHandler(createNotification))
	auth.PATCH("/notification/:id", commonHandler(updateNotification))
	auth.POST("/batch-delete/notification", commonHandler(batchDeleteNotification))

	auth.GET("/ddns", commonHandler(listDDNS))
	auth.GET("/ddns/providers", commonHandler(listProviders))
	auth.POST("/ddns", commonHandler(createDDNS))
	auth.PATCH("/ddns/:id", commonHandler(updateDDNS))
	auth.POST("/batch-delete/ddns", commonHandler(batchDeleteDDNS))

	r.NoRoute(fallbackToFrontend)
}

func recordPath(c *gin.Context) {
	url := c.Request.URL.String()
	for _, p := range c.Params {
		url = strings.Replace(url, p.Value, ":"+p.Key, 1)
	}
	c.Set("MatchedPath", url)
}

func newErrorResponse(err error) model.CommonResponse[any] {
	return model.CommonResponse[any]{
		Success: false,
		Error:   err.Error(),
	}
}

type handlerFunc[T any] func(c *gin.Context) (T, error)

// There are many error types in gorm, so create a custom type to represent all
// gorm errors here instead
type gormError struct {
	msg string
	a   []interface{}
}

func newGormError(format string, args ...interface{}) error {
	return &gormError{
		msg: format,
		a:   args,
	}
}

func (ge *gormError) Error() string {
	return fmt.Sprintf(ge.msg, ge.a...)
}

type wsError struct {
	msg string
	a   []interface{}
}

func newWsError(format string, args ...interface{}) error {
	return &wsError{
		msg: format,
		a:   args,
	}
}

func (we *wsError) Error() string {
	return fmt.Sprintf(we.msg, we.a...)
}

func commonHandler[T any](handler handlerFunc[T]) func(*gin.Context) {
	return func(c *gin.Context) {
		data, err := handler(c)
		if err == nil {
			c.JSON(http.StatusOK, model.CommonResponse[T]{Success: true, Data: data})
			return
		}
		switch err.(type) {
		case *gormError:
			log.Printf("NEZHA>> gorm error: %v", err)
			c.JSON(http.StatusOK, newErrorResponse(errors.New("database error")))
			return
		case *wsError:
			// Connection is upgraded to WebSocket, so c.Writer is no longer usable
			if msg := err.Error(); msg != "" {
				log.Printf("NEZHA>> websocket error: %v", err)
			}
			return
		default:
			c.JSON(http.StatusOK, newErrorResponse(err))
			return
		}
	}
}

func fallbackToFrontend(c *gin.Context) {
	if strings.HasPrefix(c.Request.URL.Path, "/api") {
		c.JSON(http.StatusOK, newErrorResponse(errors.New("404 Not Found")))
		return
	}
	if strings.HasPrefix(c.Request.URL.Path, "/dashboard") {
		stripPath := strings.TrimPrefix(c.Request.URL.Path, "/dashboard")
		localFilePath := filepath.Join("./admin-dist", stripPath)
		if _, err := os.Stat(localFilePath); err == nil {
			c.File(localFilePath)
			return
		}
		c.File("admin-dist/index.html")
		return
	}
	localFilePath := filepath.Join("user-dist", c.Request.URL.Path)
	if _, err := os.Stat(localFilePath); err == nil {
		c.File(localFilePath)
		return
	}
	c.File("user-dist/index.html")
}
