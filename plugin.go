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

	if hook.Client.Tags == nil {
		hook.Client.Tags = make(map[string]string)
	}

	setTag := func(key, value string) {
		if value == "" {
			return
		}
		hook.Client.Tags[key] = strings.ReplaceAll(value, " ", "_")
	}

	// Fallback to Antarctica if lookup fails
	if err != nil || record == nil || record.Country.IsoCode == "" {
		setTag("geo/country-code", "AQ")
		setTag("geo/country-name", "Antarctica")
		hook.Client.Gateway.Log(2, "GeoIP Plugin: lookup failed for %s, falling back to AQ", hook.Client.RemoteAddr)
		return
	}

	// Always include timezone (level 1)
	var timeZone string
	if granularityLevel >= GranularityTimezone {
		timeZone = record.Location.TimeZone
		setTag("geo/timezone", timeZone)
	}

	// Include country data (level 2)
	countryCode := record.Country.IsoCode
	countryName := record.Country.Names["en"]
	if countryName == "" {
		countryName = countryCode
	}
	// Normalize MaxMind's anonymous/reserved code to AQ fallback
	if countryCode == "--" {
		countryCode = "AQ"
		countryName = "Antarctica"
	}
	if granularityLevel >= GranularityCountry {
		setTag("geo/country-code", countryCode)
		setTag("geo/country-name", countryName)
	}

	// Include subdivision data (level 3)
	var subdivisionName string
	var subdivisionCode string
	if granularityLevel >= GranularitySubdivision && len(record.Subdivisions) > 0 {
		subdivisionCode = record.Subdivisions[0].IsoCode
		subdivisionName = record.Subdivisions[0].Names["en"]
		if subdivisionName == "" {
			subdivisionName = subdivisionCode
		}
		setTag("geo/subdivision-name", subdivisionName)
		setTag("geo/subdivision-code", subdivisionCode)

		// Also expose ISO-3166-2 region code/name (country-subdivision)
		if countryCode != "" && subdivisionCode != "" {
			setTag("geo/region-code", countryCode+"-"+subdivisionCode)
			setTag("geo/region-name", subdivisionName)
		}
	}

	// Include city data (level 4)
	var cityName string
	if granularityLevel >= GranularityCity {
		cityName = record.City.Names["en"]
		setTag("geo/city-name", cityName)
	}

	// Include postal code data (level 5)
	if granularityLevel >= GranularityPostal {
		postalCode := record.Postal.Code
		setTag("geo/postal-code", postalCode)
	}

	// Replace %country macro in realname with ISO country code
	if hook.Client.IrcState.RealName != "" && strings.Contains(hook.Client.IrcState.RealName, "%country") {
		hook.Client.IrcState.RealName = strings.Replace(hook.Client.IrcState.RealName, "%country", countryCode, -1)
	}

	hook.Client.Gateway.Log(2, "GeoIP Plugin (level %d): ip=%s country=%s region=%s city=%s", granularityLevel, hook.Client.RemoteAddr, countryCode, subdivisionCode, cityName)
}
