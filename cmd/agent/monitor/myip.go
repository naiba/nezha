package monitor

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

type geoIP struct {
	CountryCode string `json:"country_code,omitempty"`
	IP          string `json:"ip,omitempty"`
}

var ipv4Servers = []string{
	"https://api-ipv4.ip.sb/geoip",
	"https://ip4.seeip.org/geoip",
}

var ipv6Servers = []string{
	"https://ip6.seeip.org/geoip",
	"https://api-ipv6.ip.sb/geoip",
}

var cachedIP, cachedCountry string

func UpdateIP() {
	for {
		ipv4 := fetchGeoIP(ipv4Servers)
		ipv6 := fetchGeoIP(ipv6Servers)
		cachedIP = fmt.Sprintf("ip(v4:%s,v6:%s)", ipv4.IP, ipv6.IP)
		if ipv4.CountryCode != "" {
			cachedCountry = ipv4.CountryCode
		} else if ipv6.CountryCode != "" {
			cachedCountry = ipv6.CountryCode
		}
		time.Sleep(time.Minute * 10)
	}
}

func fetchGeoIP(servers []string) geoIP {
	var ip geoIP
	for i := 0; i < len(servers); i++ {
		resp, err := http.Get(servers[i])
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
