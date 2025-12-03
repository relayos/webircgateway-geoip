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
cp webircgateway-geoip/*.gz webircgateway/
cp webircgateway-geoip/mmdb-manager.sh webircgateway/

# reassemble the GeoLite2-City database
cd webircgateway && chmod +x mmdb-manager.sh && ./mmdb-manager.sh reassemble

# compile webircgateway with plugin
make
```

## Usage

Enable plugin in webircgateway config.conf:
```
[plugins]
plugins/geoip.so
```

### Configuration

The plugin supports granular data exposure control via the `GEOIP_GRANULARITY` environment variable:

```bash
# Set granularity level (1-5 or string values)
export GEOIP_GRANULARITY=country

# Start webircgateway
./webircgateway
```

**Granularity Levels** (hierarchical - lower numbers include all higher numbers):

| Level | String Values | Data Included |
|-------|---------------|---------------|
| 1 | `timezone`, `tz` | Timezone only |
| 2 | `country` | Timezone + Country |
| 3 | `subdivision`, `state`, `province` | + State/Province |
| 4 | `city` | + City |
| 5 | `postal`, `zip` | + Postal Code (full granularity) |

**Default**: Level 5 (full granularity)

## WEBIRC Flags

The plugin sets the following WEBIRC flags (passed to the IRC server):

| Flag | Description | Example |
|------|-------------|---------|
| `location/country-code` | ISO 3166-1 alpha-2 country code | `US` |
| `location/country-name` | English country name | `United States` |
| `location/city-name` | English city name | `New York` |
| `location/subdivision-name` | State/province name or code | `NY` |
| `location/postal-code` | Postal/ZIP code | `10001` |
| `location/timezone` | Time zone identifier | `America/New_York` |

These flags can be read by IRC server modules (e.g., InspIRCd's `m_webirc_metadata`) and converted to IRCv3 metadata.

## Database Management

The GeoLite2-City database (60MB) is stored in git as compressed chunks to avoid size limits:

```bash
# Chunk and compress a new database file
./mmdb-manager.sh chunk

# Reassemble database from chunks
./mmdb-manager.sh reassemble
```

The chunked files (`*.mmdb.chunk.*.gz`) are stored in git, while the full `.mmdb` file is gitignored.

## Docker Usage

See `Dockerfile.example` for a complete Docker build that:
1. Copies the compressed chunks into the container
2. Reassembles the database during build
3. Compiles webircgateway with the plugin

```bash
# Build with specific granularity
docker build -t webircgateway-geoip .
docker run -e GEOIP_GRANULARITY=country webircgateway-geoip
```

## Notes

This product includes GeoLite2 data created by MaxMind, available from
<a href="https://www.maxmind.com">https://www.maxmind.com</a>.
