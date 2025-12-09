package main

import (
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/kiwiirc/webircgateway/pkg/webircgateway"
	"github.com/oschwald/geoip2-golang"
)

var db *geoip2.Reader
var granularityLevel int

// Granularity levels: 1=timezone, 2=country, 3=subdivision, 4=city, 5=postal
const (
	GranularityTimezone    = 1
	GranularityCountry     = 2
	GranularitySubdivision = 3
	GranularityCity        = 4
	GranularityPostal      = 5
)

func Start(gateway *webircgateway.Gateway, pluginsQuit *sync.WaitGroup) {
	gateway.Log(2, "GeoIP plugin loading")
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		gateway.Log(3, err.Error())
		pluginsQuit.Done()
		return
	}

	// Read granularity level from environment variable, default to full granularity
	granularityLevel = GranularityPostal // Default: all data
	if envGranularity := os.Getenv("GEOIP_GRANULARITY"); envGranularity != "" {
		if level, err := strconv.Atoi(envGranularity); err == nil && level >= GranularityTimezone && level <= GranularityPostal {
			granularityLevel = level
		} else {
			// Support string values too
			switch strings.ToLower(envGranularity) {
			case "timezone", "tz":
				granularityLevel = GranularityTimezone
			case "country":
				granularityLevel = GranularityCountry
			case "subdivision", "state", "province":
				granularityLevel = GranularitySubdivision
			case "city":
				granularityLevel = GranularityCity
			case "postal", "zip":
				granularityLevel = GranularityPostal
			}
		}
	}
	gateway.Log(1, "GeoIP granularity level: %d", granularityLevel)

	ipdbFileName := dir + "/" + "GeoLite2-City.mmdb"
	gateway.Log(1, "Looking for the IPDB file: "+ipdbFileName)
	db, err = geoip2.Open(ipdbFileName)
	if err != nil {
		gateway.Log(3, err.Error())
		pluginsQuit.Done()
		return
	}
	gateway.Log(1, "GeoIP DB opened")

	webircgateway.HookRegister("irc.connection.pre", hookIrcConnectionPre)
	webircgateway.HookRegister("gateway.closing", func(hook *webircgateway.HookGatewayClosing) {
		go func() {
			gateway.Log(1, "GeoIP DB closed")
			db.Close()
			pluginsQuit.Done()
		}()
	})
}

func hookIrcConnectionPre(hook *webircgateway.HookIrcConnectionPre) {
	ip := net.ParseIP(hook.Client.RemoteAddr)
	record, err := db.City(ip)

	// Fallback to Antarctica if lookup fails
	if err != nil || record == nil || record.Country.IsoCode == "" {
		setGeoTags(hook, "AQ", "Antarctica")
		return
	}

	if hook.Client.Tags == nil {
		hook.Client.Tags = make(map[string]string)
	}

	// Always include timezone (level 1)
	timeZone := record.Location.TimeZone
	if timeZone != "" && granularityLevel >= GranularityTimezone {
		hook.Client.Tags["location/timezone"] = timeZone
	}

	// Include country data (level 2)
	var countryCode, countryName string
	if granularityLevel >= GranularityCountry {
		countryCode = record.Country.IsoCode
		countryName = record.Country.Names["en"]
		if countryCode != "" {
			hook.Client.Tags["location/country-code"] = countryCode
		}
		if countryName != "" {
			hook.Client.Tags["location/country-name"] = countryName
		}
	}

	// Include subdivision data (level 3)
	var subdivisionName string
	if granularityLevel >= GranularitySubdivision && len(record.Subdivisions) > 0 {
		subdivisionName = record.Subdivisions[0].Names["en"]
		subdivisionCode := record.Subdivisions[0].IsoCode
		if subdivisionCode != "" {
			subdivisionName = subdivisionCode
		}
		if subdivisionName != "" {
			hook.Client.Tags["location/subdivision-name"] = subdivisionName
		}
	}

	// Include city data (level 4)
	var cityName string
	if granularityLevel >= GranularityCity {
		cityName = record.City.Names["en"]
		if cityName != "" {
			hook.Client.Tags["location/city-name"] = cityName
		}
	}

	// Include postal code data (level 5)
	if granularityLevel >= GranularityPostal {
		postalCode := record.Postal.Code
		if postalCode != "" {
			hook.Client.Tags["location/postal-code"] = postalCode
		}
	}

	// Set the geo/ tags for WEBIRC (always set these for IRC integration)
	code := record.Country.IsoCode
	name := record.Country.Names["en"]
	if name == "" {
		name = code
	}
	setGeoTags(hook, code, name)

	hook.Client.Gateway.Log(2, "GeoIP Plugin (level %d): %s/%s, %s", granularityLevel, countryCode, subdivisionName, cityName)
}

func setGeoTags(hook *webircgateway.HookIrcConnectionPre, code, name string) {
	if hook.Client.Tags == nil {
		hook.Client.Tags = make(map[string]string)
	}

	// Keep tag values space-safe for WEBIRC flags
	safeCode := strings.ReplaceAll(code, " ", "_")
	safeName := strings.ReplaceAll(name, " ", "_")

	hook.Client.Tags["geo/country-code"] = safeCode
	hook.Client.Tags["geo/country"] = safeName
}
