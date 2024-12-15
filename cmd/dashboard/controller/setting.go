package controller

import (
	"errors"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/nezhahq/nezha/model"
	"github.com/nezhahq/nezha/service/singleton"
)

// List settings
// @Summary List settings
// @Schemes
// @Description List settings
// @Security BearerAuth
// @Tags common
// @Produce json
// @Success 200 {object} model.CommonResponse[model.SettingResponse]
// @Router /setting [get]
func listConfig(c *gin.Context) (model.SettingResponse, error) {
	_, isMember := c.Get(model.CtxKeyAuthorizedUser)
	authorized := isMember // TODO || isViewPasswordVerfied

	conf := model.SettingResponse{
		Config:            *singleton.Conf,
		Version:           singleton.Version,
		FrontendTemplates: singleton.FrontendTemplates,
	}

	if !authorized {
		conf = model.SettingResponse{
			Config: model.Config{
				SiteName:            conf.SiteName,
				Language:            conf.Language,
				CustomCode:          conf.CustomCode,
				CustomCodeDashboard: conf.CustomCodeDashboard,
			},
		}
	}

	conf.Config.Language = strings.Replace(conf.Config.Language, "_", "-", -1)

	return conf, nil
}

// Edit config
// @Summary Edit config
// @Security BearerAuth
// @Schemes
// @Description Edit config
// @Tags auth required
// @Accept json
// @Param body body model.SettingForm true "SettingForm"
// @Produce json
// @Success 200 {object} model.CommonResponse[any]
// @Router /setting [patch]
func updateConfig(c *gin.Context) (any, error) {
	var sf model.SettingForm
	if err := c.ShouldBindJSON(&sf); err != nil {
		return nil, err
	}
	var userTemplateValid bool
	for _, v := range singleton.FrontendTemplates {
		if !userTemplateValid && v.Path == sf.UserTemplate && !v.IsAdmin {
			userTemplateValid = true
		}
		if userTemplateValid {
			break
		}
	}
	if !userTemplateValid {
		return nil, errors.New("invalid user template")
	}

	singleton.Conf.Language = strings.Replace(sf.Language, "-", "_", -1)

	singleton.Conf.EnableIPChangeNotification = sf.EnableIPChangeNotification
	singleton.Conf.EnablePlainIPInNotification = sf.EnablePlainIPInNotification
	singleton.Conf.Cover = sf.Cover
	singleton.Conf.InstallHost = sf.InstallHost
	singleton.Conf.IgnoredIPNotification = sf.IgnoredIPNotification
	singleton.Conf.IPChangeNotificationGroupID = sf.IPChangeNotificationGroupID
	singleton.Conf.SiteName = sf.SiteName
	singleton.Conf.DNSServers = sf.DNSServers
	singleton.Conf.CustomCode = sf.CustomCode
	singleton.Conf.CustomCodeDashboard = sf.CustomCodeDashboard
	singleton.Conf.RealIPHeader = sf.RealIPHeader
	singleton.Conf.TLS = sf.TLS
	singleton.Conf.UserTemplate = sf.UserTemplate

	if err := singleton.Conf.Save(); err != nil {
		return nil, newGormError("%v", err)
	}

	singleton.OnNameserverUpdate()
	singleton.OnUpdateLang(singleton.Conf.Language)
	return nil, nil
}
