# Shodone

Let's use **Shod**​an in **one**.

Shodone could help you to manage your [shodan](<https://shodan.io/>)
API Keys, and to forward your search request.

> [!NOTE]
> You should consider Shodone as WIP, even though it can meet my simple needs right now.

## Get Started

``` shell
make run
```

You will get `shodone` binary file and it starts an HTTP server on
`localhost:8080`, which helps you to manage API keys and to forward the
search queries.

### Query example

You should get the following response by
`curl http://localhost:8080/api/shodan/host/search/filters`:

``` shell
["all", "asn", "bitcoin.ip", "bitcoin.ip_count", "bitcoin.port", "bitcoin.version", "city", "cloud.provider", "cloud.region", "cloud.service", "country", "cpe", "device", "geo", "has_ipv6", "has_screenshot", "has_ssl", "has_vuln", "hash", "hostname", "http.component", "http.component_category", "http.favicon.hash", "http.headers_hash", "http.html", "http.html_hash", "http.robots_hash", "http.securitytxt", "http.status", "http.title", "http.waf", "ip", "isp", "link", "net", "ntp.ip", "ntp.ip_count", "ntp.more", "ntp.port", "org", "os", "port", "postal", "product", "region", "scan", "screenshot.hash", "screenshot.label", "shodan.module", "snmp.contact", "snmp.location", "snmp.name", "ssh.hassh", "ssh.type", "ssl", "ssl.alpn", "ssl.cert.alg", "ssl.cert.expired", "ssl.cert.extension", "ssl.cert.fingerprint", "ssl.cert.issuer.cn", "ssl.cert.pubkey.bits", "ssl.cert.pubkey.type", "ssl.cert.serial", "ssl.cert.subject.cn", "ssl.chain_count", "ssl.cipher.bits", "ssl.cipher.name", "ssl.cipher.version", "ssl.ja3s", "ssl.jarm", "ssl.version", "state", "tag", "telnet.do", "telnet.dont", "telnet.option", "telnet.will", "telnet.wont", "version", "vuln"]
```

## Routes

| method | path | description |
|----|----|----|
| GET | `/health` | health-check of shodone |
| GET | `/config/` | get the configurations |
| PUT | `/config/api-host` | set the api host |
| GET | `/keys/` | get all keys |
| POST | `/keys/` | add a new key |
| GET | `/keys/:id` | get a specific key by id |
| DELETE | `/keys/:id` | delete a specific key by id |
| PUT | `/keys/:id` | update the status of a specific key by id |
| GET | `/keys/refresh` | refresh the status of all keys |
| ANY | `/api/*path*params` | forward the search queries with path and parameters |

## Debug

- You can use `GIN_MODE=debug` to enable GIN debug mode.
- You can use `SHODONE_LOG_LEVEL=debug` to check the debug logging.

## Postscript

This project stemmed from a minor requirement in my research process and
served as an opportunity for me to learn Golang. I hope it proves useful
to others with similar needs.
