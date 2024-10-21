package controller

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/naiba/nezha/model"
	"github.com/naiba/nezha/service/singleton"
	"golang.org/x/net/idna"
)

// Add DDNS configuration
// @Summary Add DDNS configuration
// @Security BearerAuth
// @Schemes
// @Description Add DDNS configuration
// @Tags auth required
// @Accept json
// @param request body model.DDNSForm true "DDNS Request"
// @Produce json
// @Success 200 {object} model.CommonResponse[any]
// @Router /ddns [post]
func newDDNS(c *gin.Context) {
	var df model.DDNSForm
	var p model.DDNSProfile
	var err error

	defer func() {
		if err != nil {
			c.JSON(http.StatusOK, genericErrorMsg(err))
		}
	}()

	if err = c.ShouldBindJSON(&df); err != nil {
		return
	}

	if df.MaxRetries < 1 || df.MaxRetries > 10 {
		err = errors.New("重试次数必须为大于 1 且不超过 10 的整数")
		return
	}

	p.Name = df.Name
	enableIPv4 := df.EnableIPv4 == "on"
	enableIPv6 := df.EnableIPv6 == "on"
	p.EnableIPv4 = &enableIPv4
	p.EnableIPv6 = &enableIPv6
	p.MaxRetries = df.MaxRetries
	p.Provider = df.Provider
	p.DomainsRaw = df.DomainsRaw
	p.Domains = strings.Split(p.DomainsRaw, ",")
	p.AccessID = df.AccessID
	p.AccessSecret = df.AccessSecret
	p.WebhookURL = df.WebhookURL
	p.WebhookMethod = df.WebhookMethod
	p.WebhookRequestType = df.WebhookRequestType
	p.WebhookRequestBody = df.WebhookRequestBody
	p.WebhookHeaders = df.WebhookHeaders

	for n, domain := range p.Domains {
		// IDN to ASCII
		domainValid, domainErr := idna.Lookup.ToASCII(domain)
		if domainErr != nil {
			err = fmt.Errorf("域名 %s 解析错误: %v", domain, domainErr)
			return
		}
		p.Domains[n] = domainValid
	}

	if err = singleton.DB.Create(&p).Error; err != nil {
		return
	}

	singleton.OnDDNSUpdate()
	c.JSON(http.StatusOK, model.Response{
		Code: http.StatusOK,
	})
}

// Edit DDNS configuration
// @Summary Edit DDNS configuration
// @Security BearerAuth
// @Schemes
// @Description Edit DDNS configuration
// @Tags auth required
// @Accept json
// @param request body model.DDNSForm true "DDNS Request"
// @Produce json
// @Success 200 {object} model.CommonResponse[any]
// @Router /ddns/{id} [patch]
func editDDNS(c *gin.Context) {
	var df model.DDNSForm
	var p model.DDNSProfile
	var err error

	defer func() {
		if err != nil {
			c.JSON(http.StatusOK, genericErrorMsg(err))
		}
	}()

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		return
	}

	if err = c.ShouldBindJSON(&df); err != nil {
		return
	}

	if df.MaxRetries < 1 || df.MaxRetries > 10 {
		err = errors.New("重试次数必须为大于 1 且不超过 10 的整数")
		return
	}

	p.Name = df.Name
	p.ID = id
	enableIPv4 := df.EnableIPv4 == "on"
	enableIPv6 := df.EnableIPv6 == "on"
	p.EnableIPv4 = &enableIPv4
	p.EnableIPv6 = &enableIPv6
	p.MaxRetries = df.MaxRetries
	p.Provider = df.Provider
	p.DomainsRaw = df.DomainsRaw
	p.Domains = strings.Split(p.DomainsRaw, ",")
	p.AccessID = df.AccessID
	p.AccessSecret = df.AccessSecret
	p.WebhookURL = df.WebhookURL
	p.WebhookMethod = df.WebhookMethod
	p.WebhookRequestType = df.WebhookRequestType
	p.WebhookRequestBody = df.WebhookRequestBody
	p.WebhookHeaders = df.WebhookHeaders

	for n, domain := range p.Domains {
		// IDN to ASCII
		domainValid, domainErr := idna.Lookup.ToASCII(domain)
		if domainErr != nil {
			err = fmt.Errorf("域名 %s 解析错误: %v", domain, domainErr)
			return
		}
		p.Domains[n] = domainValid
	}

	if err = singleton.DB.Save(&p).Error; err != nil {
		return
	}

	singleton.OnDDNSUpdate()
	c.JSON(http.StatusOK, model.Response{
		Code: http.StatusOK,
	})
}

// Batch delete DDNS configurations
// @Summary Batch delete DDNS configurations
// @Security BearerAuth
// @Schemes
// @Description Batch delete DDNS configurations
// @Tags auth required
// @Accept json
// @param request body []uint64 true "id list"
// @Produce json
// @Success 200 {object} model.CommonResponse[any]
// @Router /batch-delete/ddns [post]
func batchDeleteDDNS(c *gin.Context) {
	var ddnsConfigs []uint64
	var err error

	defer func() {
		if err != nil {
			c.JSON(http.StatusOK, genericErrorMsg(err))
		}
	}()

	err = c.ShouldBindJSON(&ddnsConfigs)
	if err != nil {
		return
	}

	err = singleton.DB.Unscoped().Delete(&model.DDNSProfile{}, "id in (?)", ddnsConfigs).Error
	if err != nil {
		return
	}

	singleton.OnDDNSUpdate()
	c.JSON(http.StatusOK, model.CommonResponse[interface{}]{
		Success: true,
	})
}

// TODO
func listDDNS(c *gin.Context) {}
