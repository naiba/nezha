package controller

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	GitHubAPI "github.com/google/go-github/github"
	"golang.org/x/oauth2"

	"github.com/naiba/nezha/model"
	"github.com/naiba/nezha/pkg/mygin"
	"github.com/naiba/nezha/pkg/utils"
	"github.com/naiba/nezha/service/dao"
)

type oauth2controller struct {
	oauth2Config *oauth2.Config
	r            gin.IRoutes
}

func (oa *oauth2controller) serve() {
	oa.r.GET("/oauth2/login", oa.login)
	oa.r.GET("/oauth2/callback", oa.callback)
}

func (oa *oauth2controller) login(c *gin.Context) {
	state := utils.RandStringBytesMaskImprSrcUnsafe(6)
	dao.Cache.Set(fmt.Sprintf("%s%s", model.CacheKeyOauth2State, c.ClientIP()), state, 0)
	url := oa.oauth2Config.AuthCodeURL(state, oauth2.AccessTypeOnline)
	c.Redirect(http.StatusFound, url)
}

func (oa *oauth2controller) callback(c *gin.Context) {
	// 验证登录跳转时的 State
	state, ok := dao.Cache.Get(fmt.Sprintf("%s%s", model.CacheKeyOauth2State, c.ClientIP()))
	if !ok || state.(string) != c.Query("state") {
		mygin.ShowErrorPage(c, mygin.ErrInfo{
			Code:  http.StatusBadRequest,
			Title: "登录失败",
			Msg:   fmt.Sprintf("错误信息：%s", "非法的登录方式"),
		}, true)
		return
	}
	// 拉取验证用户信息
	ctx := context.Background()
	otk, err := oa.oauth2Config.Exchange(ctx, c.Query("code"))
	if err != nil {
		mygin.ShowErrorPage(c, mygin.ErrInfo{
			Code:  http.StatusBadRequest,
			Title: "登录失败",
			Msg:   fmt.Sprintf("错误信息：%s", err),
		}, true)
		return
	}
	oc := oa.oauth2Config.Client(ctx, otk)
	client := GitHubAPI.NewClient(oc)
	gu, _, err := client.Users.Get(ctx, "")
	if err != nil {
		mygin.ShowErrorPage(c, mygin.ErrInfo{
			Code:  http.StatusBadRequest,
			Title: "登录失败",
			Msg:   fmt.Sprintf("错误信息：%s", err),
		}, true)
		return
	}
	var isAdmin bool
	if gu.GetID() > 0 {
		for _, admin := range strings.Split(dao.Conf.GitHub.Admin, ",") {
			if fmt.Sprintf("%d", gu.GetID()) == admin {
				isAdmin = true
				break
			}
		}
	}
	if !isAdmin {
		mygin.ShowErrorPage(c, mygin.ErrInfo{
			Code:  http.StatusBadRequest,
			Title: "登录失败",
			Msg:   fmt.Sprintf("错误信息：%s", "该用户不是本站点管理员，无法登录"),
		}, true)
		return
	}
	user := model.NewUserFromGitHub(gu)
	user.IssueNewToken()
	dao.DB.Save(&user)
	c.SetCookie(dao.Conf.Site.CookieName, user.Token, 60*60*24, "", "", false, false)
	c.Status(http.StatusOK)
	c.Writer.WriteString("<script>window.location.href='/'</script>")
}
