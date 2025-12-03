package main

import (
	"net"
	"os"
	"path/filepath"
	"sync"

	"github.com/kiwiirc/webircgateway/pkg/webircgateway"
	"github.com/oschwald/geoip2-golang"
)

var db *geoip2.Reader

func Start(gateway *webircgateway.Gateway, pluginsQuit *sync.WaitGroup) {
	gateway.Log(2, "GeoIP plugin loading")
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		gateway.Log(3, err.Error())
		pluginsQuit.Done()
		return
	}

	ipdbFileName := dir + "/" + "GeoLite2-Country.mmdb"
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
	if err != nil {
		hook.Client.Log(3, "Cannot find information about IP: "+ip.String())
		hook.Client.Log(3, err.Error())
		return
	}

	countryCode := record.Country.IsoCode
	countryName := record.Country.Names["en"]

	hook.Client.Gateway.Log(2, "GeoIP Plugin: %s (%s)", countryCode, countryName)

	if hook.Client.Tags == nil {
		hook.Client.Tags = make(map[string]string)
	}

	if countryCode != "" {
		hook.Client.Tags["location/country-code"] = countryCode
	}
	if countryName != "" {
		hook.Client.Tags["location/country-name"] = countryName
	}
}
