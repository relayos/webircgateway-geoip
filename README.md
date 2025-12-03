# GeoIP plugin for Webircgateway

See kiwiirc/webircgateway https://github.com/kiwiirc/webircgateway

## Installation

```bash
# first clone webircgateway
git clone https://github.com/kiwiirc/webircgateway.git

# clone plugin
git clone https://github.com/relayos/webircgateway-geoip.git

# create folder for geoip plugin
mkdir webircgateway/plugins/geoip

# copy plugin files
cp webircgateway-geoip/plugin.go webircgateway/plugins/geoip/
cp webircgateway-geoip/GeoLite2-Country.mmdb webircgateway/GeoLite2-Country.mmdb

# compile webircgateway with plugin
cd webircgateway && make
```

## Usage

Enable plugin in webircgateway config.conf:
```
[plugins]
plugins/geoip.so
```

## WEBIRC Flags

The plugin sets the following WEBIRC flags (passed to the IRC server):

| Flag | Description | Example |
|------|-------------|---------|
| `location/country-code` | ISO 3166-1 alpha-2 country code | `US` |
| `location/country-name` | English country name | `United States` |

These flags can be read by IRC server modules (e.g., InspIRCd's `m_webirc_metadata`) and converted to IRCv3 metadata.

## Notes

This product includes GeoLite2 data created by MaxMind, available from
<a href="https://www.maxmind.com">https://www.maxmind.com</a>.
