package mygin

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/naiba/nezha/model"
	"github.com/naiba/nezha/pkg/utils"
	"github.com/naiba/nezha/service/singleton"
)

func PreferredTheme(c *gin.Context) {
	// 采用前端传入的主题
	if theme, err := c.Cookie("preferred_theme"); err == nil {
		if _, has := model.Themes[theme]; has {
			// 检验自定义主题
			if theme == "custom" && singleton.Conf.Site.Theme != "custom" && !utils.IsFileExists("resource/template/theme-custom/home.html") {
				return
			}
			c.Set(model.CtxKeyPreferredTheme, theme)
		}
	}
}

func GetPreferredTheme(c *gin.Context, path string) string {
	if theme, has := c.Get(model.CtxKeyPreferredTheme); has {
		return fmt.Sprintf("theme-%s%s", theme, path)
	}
	return fmt.Sprintf("theme-%s%s", singleton.Conf.Site.Theme, path)
}
