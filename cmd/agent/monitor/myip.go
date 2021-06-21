package monitor

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/naiba/nezha/pkg/utils"
)

type geoIP struct {
	CountryCode string `json:"country_code,omitempty"`
	IP          string `json:"ip,omitempty"`
}

var (
	ipv4Servers = []string{
		"https://api-ipv4.ip.sb/geoip",
		"https://ip4.seeip.org/geoip",
		"https://ipapi.co/json",
	}
	ipv6Servers = []string{
		"https://ip6.seeip.org/geoip",
		"https://api-ipv6.ip.sb/geoip",
		"https://ipapi.co/json",
	}
	cachedIP, cachedCountry string
	httpClientV4            = utils.NewSingleStackHTTPClient(time.Second*20, time.Second*5, time.Second*10, false)
	httpClientV6            = utils.NewSingleStackHTTPClient(time.Second*20, time.Second*5, time.Second*10, true)
)

func UpdateIP() {
	for {
		ipv4 := fetchGeoIP(ipv4Servers, false)
		ipv6 := fetchGeoIP(ipv6Servers, true)
		cachedIP = fmt.Sprintf("ip(v4:%s,v6:[%s])", ipv4.IP, ipv6.IP)
		if ipv4.CountryCode != "" {
			cachedCountry = ipv4.CountryCode
		} else if ipv6.CountryCode != "" {
			cachedCountry = ipv6.CountryCode
		}
		time.Sleep(time.Minute * 10)
	}
}

func fetchGeoIP(servers []string, isV6 bool) geoIP {
	var ip geoIP
	var resp *http.Response
	var err error
	for i := 0; i < len(servers); i++ {
		if isV6 {
			resp, err = httpClientV6.Get(servers[i])
		} else {
			resp, err = httpClientV4.Get(servers[i])
		}
		if err == nil {
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				continue
			}
			resp.Body.Close()
			err = json.Unmarshal(body, &ip)
			if err != nil {
				continue
			}
			return ip
		}
	}
	return ip
}
