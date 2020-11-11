package mygin

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/naiba/nezha/model"
)

// ErrInfo ..
type ErrInfo struct {
	Code  uint64
	Title string
	Msg   string
	Link  string
	Btn   string
}

// ShowErrorPage ..
func ShowErrorPage(c *gin.Context, i ErrInfo, isPage bool) {
	if isPage {
		c.HTML(http.StatusOK, "page/error", CommonEnvironment(c, gin.H{
			"Code":  i.Code,
			"Title": i.Title,
			"Msg":   i.Msg,
			"Link":  i.Link,
			"Btn":   i.Btn,
		}))
	} else {
		c.JSON(http.StatusOK, model.Response{
			Code:    i.Code,
			Message: i.Msg,
		})
	}
	c.Abort()
}
