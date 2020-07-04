# DNS Monster

Passive DNS collection and monitoring built with Golang, Clickhouse and Grafana: [Blogpost](https://blog.n0p.me/dnsmonster/)

# Build Manually

Make sure you have `libpcap-devel` package installed

`go get gitlab.com/mosajjal/dnsmonster`

## Static Build (WIP)

`$ git clone https://gitlab.com/mosajjal/dnsmonster`
`$ cd dnsmonster`
`$ go get`
`$ go build --ldflags "-L /root/libpcap-1.9.1/libpcap.a -linkmode external -extldflags \"-I/usr/include/libnl3 -lnl-genl-3 -lnl-3 -static\"" -a -o dnsmonster`

# Installation Steps

* Identify the interface(s) you're trying to monitor
* Have a Clickhouse installation ready. This can be a simple Container, or a full-blown cluster
* Download and run `dnsmonster` as a service
* Check to see if you are indexing the traffic (Using ClickHouse client)
* Setup Grafana to connect to ClickHouse

# Quick start

## On the Collector node

* Clone the repo
```
git clone https://gitlab.com/mosajjal/dnsmonster.git
cd dnsmonster/setup/
```

* open `autobuild.sh` and edit `IFACE` variable with your TAP interface 

* run `autobuild.sh` NOTE: the script only works with RHEL7/CentOS7

Autobuild script does the following, you can easily replicate for different environments:
 - Installs Podman and libpcap
 - Creates data and log directories in `/data/`
 - Downloads `dnsmonster` and makes it executable in `/opt/dnsmonster/dnsmonster`
 - Sets up retention policy of 30 days for ClickHouse data in a daily Cronjob
 - Runs a ClickHouse Container and copies DNS-related lookup tables inside the container
 - Adds ClickHouse Container service to startup
 - Runs a ClickHouse client Container to import basic schemas
 - Sets up `dnsmonster` service and log rotation policies 
 - Tests the setup and exits

## On Viewer node

* On your viewer machine, run grafana.sh. The script will set up a Container running grafana and all the related plugins.

* Navigate to "add source" in Grafana and connect with your ClickHouse server

* import panel.json and set it to use your ClickHouse

* Enjoy!
