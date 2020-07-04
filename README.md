# DNS Monster

Passive DNS collection and monitoring built with Golang, Clickhouse and Grafana: [Blogpost](https://blog.n0p.me/dnsmonster/)


# Quick start

# AIO Installation using Docker

running `./autobuild.sh` now creates 3 containers:

* an instance of `dnsmonster` to look at the traffic on `lo` interface (you can change the interface from the `docker-compose.yml`)
* an instance of `clickhouse` to collect `dnsmonster`'s output and saves all the logs/data to `/opt/ch-data/` (Can be changed from `docker-compose.yml`)
* an instance of `grafana` looking at the `clickhouse` data with pre-built dashboard.

## What's the retention policy

The default retention policy for the DNS data is set to 30 days. You can change the number *BEFORE* running `./autobuild.sh` by editing `clickhouse/tables.sql`, in the line `TTL DnsDate + INTERVAL 30 DAY;`

## Scalable deployment Howto

Coming soon!

# Build Manually

Make sure you have `libpcap-devel` package installed

`go get gitlab.com/mosajjal/dnsmonster/src`

## Static Build (WIP)

```
 $ git clone https://gitlab.com/mosajjal/dnsmonster
 $ cd dnsmonster/src/
 $ go get
 $ go build --ldflags "-L /root/libpcap-1.9.1/libpcap.a -linkmode external -extldflags \"-I/usr/include/libnl3 -lnl-genl-3 -lnl-3 -static\"" -a -o dnsmonster
```

## pre-built Binary

The latest version will be available here:`n0p.me/bin/dnsmonster`
