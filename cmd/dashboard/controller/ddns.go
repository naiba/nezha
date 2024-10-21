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
func newDDNS(c *gin.Context) error {
	var df model.DDNSForm
	var p model.DDNSProfile

	if err := c.ShouldBindJSON(&df); err != nil {
		return err
	}

	if df.MaxRetries < 1 || df.MaxRetries > 10 {
		return errors.New("重试次数必须为大于 1 且不超过 10 的整数")
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
			return fmt.Errorf("域名 %s 解析错误: %v", domain, domainErr)
		}
		p.Domains[n] = domainValid
	}

	if err := singleton.DB.Create(&p).Error; err != nil {
		return newGormError("%v", err)
	}

	singleton.OnDDNSUpdate()
	c.JSON(http.StatusOK, model.Response{
		Code: http.StatusOK,
	})
	return nil
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
func editDDNS(c *gin.Context) error {
	var df model.DDNSForm
	var p model.DDNSProfile

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		return err
	}

	if err := c.ShouldBindJSON(&df); err != nil {
		return err
	}

	if df.MaxRetries < 1 || df.MaxRetries > 10 {
		return errors.New("重试次数必须为大于 1 且不超过 10 的整数")
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
			return fmt.Errorf("域名 %s 解析错误: %v", domain, domainErr)
		}
		p.Domains[n] = domainValid
	}

	if err = singleton.DB.Save(&p).Error; err != nil {
		return newGormError("%v", err)
	}

	singleton.OnDDNSUpdate()
	c.JSON(http.StatusOK, model.Response{
		Code: http.StatusOK,
	})
	return nil
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
func batchDeleteDDNS(c *gin.Context) error {
	var ddnsConfigs []uint64

	if err := c.ShouldBindJSON(&ddnsConfigs); err != nil {
		return err
	}

	if err := singleton.DB.Unscoped().Delete(&model.DDNSProfile{}, "id in (?)", ddnsConfigs).Error; err != nil {
		return newGormError("%v", err)
	}

	singleton.OnDDNSUpdate()
	c.JSON(http.StatusOK, model.CommonResponse[interface{}]{
		Success: true,
	})
	return nil
}

// TODO
func listDDNS(c *gin.Context) {}
