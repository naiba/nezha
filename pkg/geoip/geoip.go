package geoip

import (
	"embed"
	"fmt"
	"net"
	"strings"

	maxminddb "github.com/oschwald/maxminddb-golang"
)

//go:embed geoip.db
var geoDBFS embed.FS

type IPInfo struct {
	Country       string `maxminddb:"country"`
	CountryName   string `maxminddb:"country_name"`
	Continent     string `maxminddb:"continent"`
	ContinentName string `maxminddb:"continent_name"`
}

func Lookup(ip net.IP, record *IPInfo) (string, error) {
	dbFile, err := geoDBFS.ReadFile("geoip.db")
	if err != nil {
		return "", err
	}

	db, err := maxminddb.FromBytes(dbFile)
	if err != nil {
		return "", err
	}
	defer db.Close()

	err = db.Lookup(ip, record)
	if err != nil {
		return "", err
	}

	if record.Country != "" {
		return strings.ToLower(record.Country), nil
	} else if record.Continent != "" {
		return strings.ToLower(record.Continent), nil
	}

	return "", fmt.Errorf("IP not found")
}
