package monitor

import (
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/naiba/nezha/pkg/utils"
)

func TestGeoIPApi(t *testing.T) {
	for i := 0; i < len(geoIPApiList); i++ {
		resp, err := httpGetWithUA(httpClientV4, geoIPApiList[i])
		assert.Nil(t, err)
		body, err := ioutil.ReadAll(resp.Body)
		assert.Nil(t, err)
		resp.Body.Close()
		var ip geoIP
		err = ip.Unmarshal(body)
		assert.Nil(t, err)
		t.Logf("%s %s %s", geoIPApiList[i], ip.CountryCode, utils.IPDesensitize(ip.IP))
		assert.True(t, ip.IP != "")
		assert.True(t, ip.CountryCode != "")
	}
}

func TestFetchGeoIP(t *testing.T) {
	ip := fetchGeoIP(geoIPApiList, false)
	assert.NotEmpty(t, ip.IP)
	assert.NotEmpty(t, ip.CountryCode)
}
