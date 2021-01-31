package controller

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"golang.org/x/crypto/bcrypt"

	"github.com/naiba/nezha/model"
	"github.com/naiba/nezha/pkg/mygin"
	"github.com/naiba/nezha/service/dao"
)

type commonPage struct {
	r *gin.Engine
}

func (cp *commonPage) serve() {
	cr := cp.r.Group("")
	cr.Use(mygin.Authorize(mygin.AuthorizeOption{}))
	cr.POST("/view-password", cp.issueViewPassword)
	cr.Use(cp.checkViewPassword) // 前端查看密码鉴权
	cr.GET("/", cp.home)
	cr.GET("/service", cp.service)
	cr.GET("/ws", cp.ws)
}

type viewPasswordForm struct {
	Password string
}

func (p *commonPage) issueViewPassword(c *gin.Context) {
	var vpf viewPasswordForm
	err := c.ShouldBind(&vpf)
	var hash []byte
	if err == nil && vpf.Password != dao.Conf.Site.ViewPassword {
		err = errors.New("查看密码错误")
	}
	if err == nil {
		hash, err = bcrypt.GenerateFromPassword([]byte(vpf.Password), bcrypt.DefaultCost)
	}
	if err != nil {
		mygin.ShowErrorPage(c, mygin.ErrInfo{
			Title: "出现错误",
			Msg:   fmt.Sprintf("请求错误：%s", err),
		}, true)
		c.Abort()
		return
	}
	c.SetCookie(dao.Conf.Site.CookieName+"-vp", string(hash), 60*60*24, "", "", false, false)
	c.Redirect(http.StatusFound, c.Request.Referer())
}

func (p *commonPage) checkViewPassword(c *gin.Context) {
	if dao.Conf.Site.ViewPassword == "" {
		c.Next()
		return
	}
	if _, authorized := c.Get(model.CtxKeyAuthorizedUser); authorized {
		c.Next()
		return
	}

	// 验证查看密码
	viewPassword, _ := c.Cookie(dao.Conf.Site.CookieName + "-vp")
	if err := bcrypt.CompareHashAndPassword([]byte(viewPassword), []byte(dao.Conf.Site.ViewPassword)); err != nil {
		c.HTML(http.StatusOK, "theme-"+dao.Conf.Site.Theme+"/viewpassword", mygin.CommonEnvironment(c, gin.H{
			"Title":      "验证查看密码",
			"CustomCode": dao.Conf.Site.CustomCode,
		}))
		c.Abort()
		return
	}

	c.Next()
}

type ServiceItem struct {
	Monitor     model.Monitor
	TotalUp     uint64
	TotalDown   uint64
	CurrentUp   uint64
	CurrentDown uint64
	Delay       *[30]float32
	Up          *[30]int
	Down        *[30]int
}

func (p *commonPage) service(c *gin.Context) {
	var msm map[uint64]*ServiceItem

	var cached bool
	if _, has := c.Get(model.CtxKeyAuthorizedUser); !has {
		data, has := dao.Cache.Get(model.CacheKeyServicePage)
		if has {
			log.Println("use cache")
			msm = data.(map[uint64]*ServiceItem)
			cached = true
		}
	}

	if !cached {
		msm = make(map[uint64]*ServiceItem)
		var ms []model.Monitor
		dao.DB.Find(&ms)
		year, month, day := time.Now().Date()
		today := time.Date(year, month, day, 0, 0, 0, 0, time.Local)
		var mhs []model.MonitorHistory
		dao.DB.Where("created_at >= ?", today.AddDate(0, 0, -29)).Find(&mhs)

		for i := 0; i < len(ms); i++ {
			msm[ms[i].ID] = &ServiceItem{
				Monitor: ms[i],
				Delay:   &[30]float32{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
				Up:      &[30]int{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
				Down:    &[30]int{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			}
		}
		// 整合数据
		todayStatus := make(map[uint64][]bool)
		for i := 0; i < len(mhs); i++ {
			dayIndex := 29
			if mhs[i].CreatedAt.Before(today) {
				dayIndex = 28 - (int(today.Sub(mhs[i].CreatedAt).Hours()) / 24)
			} else {
				todayStatus[mhs[i].MonitorID] = append(todayStatus[mhs[i].MonitorID], mhs[i].Successful)
			}
			if mhs[i].Successful {
				msm[mhs[i].MonitorID].TotalUp++
				msm[mhs[i].MonitorID].Delay[dayIndex] = (msm[mhs[i].MonitorID].Delay[dayIndex]*float32(msm[mhs[i].MonitorID].Up[dayIndex]) + mhs[i].Delay) / float32(msm[mhs[i].MonitorID].Up[dayIndex]+1)
				msm[mhs[i].MonitorID].Up[dayIndex]++
			} else {
				msm[mhs[i].MonitorID].TotalDown++
				msm[mhs[i].MonitorID].Down[dayIndex]++
			}
		}
		// 当日最后 20 个采样作为当前状态
		for _, m := range msm {
			for i := len(todayStatus[m.Monitor.ID]) - 1; i >= 0 && i >= (len(todayStatus[m.Monitor.ID])-1-20); i-- {
				if todayStatus[m.Monitor.ID][i] {
					m.CurrentUp++
				} else {
					m.CurrentDown++
				}
			}
		}
		// 未登录人员缓存十分钟
		dao.Cache.Set(model.CacheKeyServicePage, msm, time.Minute*10)
	}

	c.HTML(http.StatusOK, "theme-"+dao.Conf.Site.Theme+"/service", mygin.CommonEnvironment(c, gin.H{
		"Title":      "服务状态",
		"Services":   msm,
		"CustomCode": dao.Conf.Site.CustomCode,
	}))
}

func (cp *commonPage) home(c *gin.Context) {
	dao.SortedServerLock.RLock()
	defer dao.SortedServerLock.RUnlock()

	c.HTML(http.StatusOK, "theme-"+dao.Conf.Site.Theme+"/home", mygin.CommonEnvironment(c, gin.H{
		"Servers":    dao.SortedServerList,
		"CustomCode": dao.Conf.Site.CustomCode,
	}))
}

var upgrader = websocket.Upgrader{}

func (cp *commonPage) ws(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		mygin.ShowErrorPage(c, mygin.ErrInfo{
			Code:  http.StatusInternalServerError,
			Title: "网络错误",
			Msg:   "Websocket协议切换失败",
			Link:  "/",
			Btn:   "返回首页",
		}, true)
		return
	}
	defer conn.Close()
	for {
		dao.SortedServerLock.RLock()
		err = conn.WriteJSON(dao.SortedServerList)
		dao.SortedServerLock.RUnlock()
		if err != nil {
			break
		}
		time.Sleep(time.Second * 2)
	}
}
