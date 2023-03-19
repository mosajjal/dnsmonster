![Build Status](https://github.com/mosajjal/dnsmonster/workflows/Build%20Test/badge.svg?style=flat-square)
![Go Version](https://img.shields.io/github/go-mod/go-version/mosajjal/dnsmonster/main?filename=go.mod&style=flat-square)
![Latest Version](https://img.shields.io/github/v/tag/mosajjal/dnsmonster?label=latest&style=flat-square)
![License](https://img.shields.io/github/license/mosajjal/dnsmonster?style=flat-square)
![Open Issues](https://img.shields.io/github/issues/mosajjal/dnsmonster?style=flat-square)

![Logo](static/dnsmonster-logo.svg)

Table of Contents
- [Dnsmonster](#dnsmonster)
- [Main features](#main-features)
- [Installation](#installation)
  - [Linux](#linux)
    - [Container](#container)
    - [Build manually](#build-manually)
    - [Build statically](#build-statically)
  - [Windows](#windows)
  - [FreeBSD and MacOS](#freebsd-and-macos)
- [Architecture](#architecture)
  - [All-in-one Installation using Docker](#all-in-one-installation-using-docker)
    - [All-in-one Demo](#all-in-one-demo)
  - [Enterprise Deployment](#enterprise-deployment)
- [Configuration](#configuration)
  - [Command line options](#command-line-options)
  - [Environment variables](#environment-variables)
  - [Configuration file](#configuration-file)
  - [What's the retention policy](#whats-the-retention-policy)
- [Sampling and Skipping](#sampling-and-skipping)
  - [pre-process sampling](#pre-process-sampling)
  - [skip domains](#skip-domains)
  - [allow domains](#allow-domains)
  - [SAMPLE in clickhouse SELECT queries](#sample-in-clickhouse-select-queries)
- [Supported Inputs](#supported-inputs)
- [Supported Outputs](#supported-outputs)
- [Roadmap](#roadmap)
- [Related projects](#related-projects)

# Dnsmonster

Passive DNS monitoring framework built on Golang. 
`dnsmonster` implements a packet sniffer for DNS traffic. It can accept traffic from a `pcap` file, a live interface or a `dnstap` socket, 
and can be used to index and store hundreds of thousands of DNS queries per second as it has shown to be capable of indexing 200k+ DNS queries per second on a commodity computer. It aims to be scalable, simple, and easy to use, and help
security teams to understand the details about an enterprise's DNS traffic. `dnsmonster` doesn't look to follow DNS conversations, rather it aims to index DNS packets as soon as they come in. It also doesn't aim to breach
the privacy of the end-users, with the ability to mask Layer 3 IPs (IPv4 and IPv6), enabling teams to perform trend analysis on aggregated data without being able to trace back the queries to an individual. [Blogpost](https://blog.n0p.me/dnsmonster/)

The code before version 1.x is considered beta quality and is subject to breaking changes. Please visit the release notes for each tag to see the list of breaking scenarios between each release, and how to mitigate potential data loss.

![Inner logic of dnsmonster](static/dnsmonster-inner.svg)

# Main features

- Ability to use Linux's `afpacket` and zero-copy packet capture.
- Supports BPF
- Ability to mask IP address to enhance privacy
- Ability to have a pre-processing sampling ratio
- Ability to have a list of "skip" `fqdn`s to avoid writing some domains/suffix/prefix to storage
- Ability to have a list of "allow" domains, used to log access to certain domains
- Hot-reload of skip and allow domain files/urls
- Modular output with configurable logic per output stream.
- Automatic data retention policy using ClickHouse's TTL attribute
- Built-in Grafana dashboard for ClickHouse output.
- Ability to be shipped as a single, statically linked binary
- Ability to be configured using environment variables, command line options or configuration file
- Ability to sample outputs using ClickHouse's SAMPLE capability
- Ability to send metrics using `prometheus` and `statstd`
- High compression ratio thanks to ClickHouse's built-in LZ4 storage
- Supports DNS Over TCP, Fragmented DNS (udp/tcp) and IPv6
- Supports [dnstap](https://github.com/dnstap/golang-dnstap) over Unix socket or TCP
- built-in SIEM integration with Splunk and Microsoft Sentinel

# Installation

## Linux

Best way to get started with `dnsmonster` is to download the binary from the release section. The binary is statically built against `musl`, hence it should work out of the box for many distros. For `afpacket` support, you must use kernel 3.x+. Any modern Linux distribution (CentOS/RHEL 7+, Ubuntu 14.0.4.2+, Debian 7+) is shipped with a 3.x+ version so it should work out of the box. If your distro isn't working with the pre-compiled version, please submit an issue with the details, and build `dnsmonster` manually using this section [Build Manually](#build-manually).

### Container

Since `dnsmonster` uses raw packet capture funcationality, Docker/Podman daemon must grant the capability to the container

```
sudo docker run --rm -it --net=host --cap-add NET_RAW --cap-add NET_ADMIN --name dnsmonster ghcr.io/mosajjal/dnsmonster:latest --devName lo --stdoutOutputType=1
```


### Build manually

- with `libpcap`:
  Make sure you have `go`, `libpcap-devel` and `linux-headers` packages installed. The name of the packages might differ based on your distribution. After this, simply clone the repository and run `go build ./cmd/dnsmonster`

```sh
git clone https://github.com/mosajjal/dnsmonster --depth 1 /tmp/dnsmonster 
cd /tmp/dnsmonster
go get
go build -o dnsmonster ./cmd/dnsmonster
```

- without `libpcap`:
`dnsmonster` only uses one function from `libpcap`, and that's converting the `tcpdump`-style filters into BPF bytecode. If you can live with no BPF support, you can build `dnsmonster` without `libpcap`. Note that for any other platform, the packet capture falls back to `libpcap` so it becomes a hard dependency (*BSD, Windows, Darwin)

```sh
git clone https://github.com/mosajjal/dnsmonster --depth 1 /tmp/dnsmonster 
cd /tmp/dnsmonster
go get
go build -o dnsmonster -tags nolibpcap ./cmd/dnsmonster
```

The above build also works on ARMv7 (RPi4) and AArch64.

### Build statically

If you have a copy of `libpcap.a`, you can build the statically link it to `dnsmonster` and build it fully statically. In the code below, please change `/root/libpcap-1.9.1/libpcap.a` to the location of your copy.

```
git clone https://github.com/mosajjal/dnsmonster --depth 1 /tmp/dnsmonster
cd /tmp/dnsmonster/
go get
go build --ldflags "-L /root/libpcap-1.9.1/libpcap.a -linkmode external -extldflags \"-I/usr/include/libnl3 -lnl-genl-3 -lnl-3 -static\"" -a -o dnsmonster ./cmd/dnsmonster
```

For more information on how the statically linked binary is created, take a look at [this](Dockerfile) Dockerfile.

## Windows

Bulding on Windows is much the same as Linux. Just make sure that you have `npcap`. Clone the repository (`--history 1` works), and run `go get` and `go build ./cmd/dnsmonster`

As mentioned, Windows release of the binary depends on [npcap](https://nmap.org/npcap/#download) to be installed. After installation, the binary should work out of the box. It's been tested in a Windows 10 environment and it executed without an issue. To find interface names to give `--devName` parameter and start sniffing, you'll need to do the following:

  - open cmd.exe as Administrator and run the following: `getmac.exe`, you'll see a table with your interfaces' MAC address and a Transport Name column with something like this: `\Device\Tcpip_{16000000-0000-0000-0000-145C4638064C}`
  - run `dnsmonster.exe` in `cmd.exe` like this:

```sh
dnsmonster.exe --devName \Device\NPF_{16000000-0000-0000-0000-145C4638064C}
```

Note that you must change `\Tcpip` from `getmac.exe` to `\NPF` and then pass it to `dnsmonster.exe`.

## FreeBSD and MacOS

Much the same as Linux and Windows, make sure you have `git`, `libpcap` and `go` installed, then follow the same instructions:

```sh
git clone https://github.com/mosajjal/dnsmonster --depth 1 /tmp/dnsmonster 
cd /tmp/dnsmonster
go get
go build -o dnsmonster ./cmd/dnsmonster
```

# Architecture

## All-in-one Installation using Docker

![Basic AIO Diagram](static/dnsmonster-basic.svg)

In the example diagram, the egress/ingress of the DNS server traffic is captured, after that, an optional layer of packet aggregation is added before hitting the DNSMonster Server. The outbound data going out of DNS Servers is quite useful to perform cache and performance analysis on the DNS fleet. If an aggregator isn't available for you, you can have both TAPs connected directly to DNSMonster and have two DNSMonster Agents looking at the traffic. 

running `./autobuild.sh` creates multiple containers:

* multiple instances of `dnsmonster` to look at the traffic on any interface. Interface list will be prompted as part of `autobuild.sh`
* an instance of `clickhouse` to collect `dnsmonster`'s output and saves all the logs/data to a data and logs directory. Both will be prompted as part of `autobuild.sh`
* an instance of `grafana` looking at the `clickhouse` data with pre-built dashboard.


### All-in-one Demo

[![AIO Demo](static/aio_demo.svg)](static/aio_demo.svg)


## Enterprise Deployment


![Basic AIO Diagram](static/dnsmonster-enterprise.svg)


# Configuration

DNSMonster can be configured using 3 different methods. Command line options, Environment variables and configuration file. Order of precedence:

- Command line options (Case-insensitive)
- Environment variables (Always upper-case)
- Configuration file (Case-sensitive, lowercase)
- Default values (No configuration)

## Command line options

Note that command line arguments are case-insensitive as of v0.9.5

[//]: <> (start of command line options)
```sh
# [capture]
# Device used to capture
--devname=

# Pcap filename to run
--pcapfile=

# dnstap socket path. Example: unix:///tmp/dnstap.sock, tcp://127.0.0.1:8080
--dnstapsocket=

# Port selected to filter packets
--port=53

# Capture Sampling by a:b. eg sampleRatio of 1:100 will process 1 percent of the incoming packets
--sampleratio=1:1

# Cleans up packet hash table used for deduplication
--dedupcleanupinterval=1m0s

# Set the dnstap socket permission, only applicable when unix:// is used
--dnstappermission=755

# Number of routines used to handle received packets
--packethandlercount=2

# Size of the tcp assembler
--tcpassemblychannelsize=10000

# Size of the tcp result channel
--tcpresultchannelsize=10000

# Number of routines used to handle tcp packets
--tcphandlercount=1

# Size of the channel to send packets to be defragged
--defraggerchannelsize=10000

# Size of the channel where the defragged packets are returned
--defraggerchannelreturnsize=10000

# Size of the packet handler channel
--packetchannelsize=1000

# Afpacket Buffersize in MB
--afpacketbuffersizemb=64

# BPF filter applied to the packet stream. If port is selected, the packets will not be defragged.
--filter=((ip and (ip[9] == 6 or ip[9] == 17)) or (ip6 and (ip6[6] == 17 or ip6[6] == 6 or ip6[6] == 44)))

# Use AFPacket for live captures. Supported on Linux 3.0+ only
--useafpacket

# The PCAP capture does not contain ethernet frames
--noetherframe

# Deduplicate incoming packets, Only supported with --devName and --pcapFile. Experimental 
--dedup

# Do not put the interface in promiscuous mode
--nopromiscuous

# [clickhouse_output]
# Address of the clickhouse database to save the results. multiple values can be provided.
--clickhouseaddress=localhost:9000

# Username to connect to the clickhouse database
--clickhouseusername=

# Password to connect to the clickhouse database
--clickhousepassword=

# Database to connect to the clickhouse database
--clickhousedatabase=default

# Interval between sending results to ClickHouse. If non-0, Batch size is ignored and batch delay is used
--clickhousedelay=0s

# Clickhouse connection LZ4 compression level, 0 means no compression
--clickhousecompress=0

# Debug Clickhouse connection
--clickhousedebug

# Use TLS for Clickhouse connection
--clickhousesecure

# Save full packet query and response in JSON format.
--clickhousesavefullquery

# What should be written to clickhouse. options:
#	0: Disable Output
#	1: Enable Output without any filters
#	2: Enable Output and apply skipdomains logic
#	3: Enable Output and apply allowdomains logic
#	4: Enable Output and apply both skip and allow domains logic
--clickhouseoutputtype=0

# Minimum capacity of the cache array used to send data to clickhouse. Set close to the queries per second received to prevent allocations
--clickhousebatchsize=100000

# Number of Clickhouse output Workers
--clickhouseworkers=1

# Channel Size for each Clickhouse Worker
--clickhouseworkerchannelsize=100000

# [elastic_output]
# What should be written to elastic. options:
#	0: Disable Output
#	1: Enable Output without any filters
#	2: Enable Output and apply skipdomains logic
#	3: Enable Output and apply allowdomains logic
#	4: Enable Output and apply both skip and allow domains logic
--elasticoutputtype=0

# elastic endpoint address, example: http://127.0.0.1:9200. Used if elasticOutputType is not none
--elasticoutputendpoint=

# elastic index
--elasticoutputindex=default

# Send data to Elastic in batch sizes
--elasticbatchsize=1000

# Interval between sending results to Elastic if Batch size is not filled
--elasticbatchdelay=1s

# [file_output]
# What should be written to file. options:
#	0: Disable Output
#	1: Enable Output without any filters
#	2: Enable Output and apply skipdomains logic
#	3: Enable Output and apply allowdomains logic
#	4: Enable Output and apply both skip and allow domains logic
--fileoutputtype=0

# Path to output folder. Used if fileOutputType is not none
--fileoutputpath=

# Interval to rotate the file in cron format
--fileOutputrotatecron=0 0 * * *

# Number of files to keep. 0 to disable rotation
--fileoutputrotatecount=4

# Output format for file. options:json,csv, csv_no_header, gotemplate. note that the csv splits the datetime format into multiple fields
--fileoutputformat=json

# Go Template to format the output as needed
--fileoutputgotemplate={{.}}

# [influx_output]
# What should be written to influx. options:
#	0: Disable Output
#	1: Enable Output without any filters
#	2: Enable Output and apply skipdomains logic
#	3: Enable Output and apply allowdomains logic
#	4: Enable Output and apply both skip and allow domains logic
--influxoutputtype=0

# influx Server address, example: http://localhost:8086. Used if influxOutputType is not none
--influxoutputserver=

# Influx Server Auth Token
--influxoutputtoken=dnsmonster

# Influx Server Bucket
--influxoutputbucket=dnsmonster

# Influx Server Org
--influxoutputorg=dnsmonster

# Minimum capacity of the cache array used to send data to Influx
--influxoutputworkers=8

# Minimum capacity of the cache array used to send data to Influx
--influxbatchsize=1000

# [kafka_output]
# What should be written to kafka. options:
#	0: Disable Output
#	1: Enable Output without any filters
#	2: Enable Output and apply skipdomains logic
#	3: Enable Output and apply allowdomains logic
#	4: Enable Output and apply both skip and allow domains logic
--kafkaoutputtype=0

# kafka broker address(es), example: 127.0.0.1:9092. Used if kafkaOutputType is not none
--kafkaoutputbroker=

# Kafka topic for logging
--kafkaoutputtopic=dnsmonster

# Minimum capacity of the cache array used to send data to Kafka
--kafkabatchsize=1000

# Output format. options:json, gob. 
--kafkaoutputformat=json

# Kafka connection timeout in seconds
--kafkatimeout=3

# Interval between sending results to Kafka if Batch size is not filled
--kafkabatchdelay=1s

# Compress Kafka connection
--kafkacompress

# Use TLS for kafka connection
--kafkasecure

# Path of CA certificate that signs Kafka broker certificate
--kafkacacertificatepath=

# Path of TLS certificate to present to broker
--kafkatlscertificatepath=

# Path of TLS certificate key
--kafkatlskeypath=

# [psql_output]
# What should be written to Microsoft Psql. options:
#	0: Disable Output
#	1: Enable Output without any filters
#	2: Enable Output and apply skipdomains logic
#	3: Enable Output and apply allowdomains logic
#	4: Enable Output and apply both skip and allow domains logic
--psqloutputtype=0

# Psql endpoint used. must be in uri format. example: postgres://username:password@hostname:port/database?sslmode=disable
--psqlendpoint=

# Number of PSQL workers
--psqlworkers=1

# Psql Batch Size
--psqlbatchsize=1

# Interval between sending results to Psql if Batch size is not filled. Any value larger than zero takes precedence over Batch Size
--psqlbatchdelay=0s

# Timeout for any INSERT operation before we consider them failed
--psqlbatchtimeout=5s

# Save full packet query and response in JSON format.
--psqlsavefullquery

# [sentinel_output]
# What should be written to Microsoft Sentinel. options:
#	0: Disable Output
#	1: Enable Output without any filters
#	2: Enable Output and apply skipdomains logic
#	3: Enable Output and apply allowdomains logic
#	4: Enable Output and apply both skip and allow domains logic
--sentineloutputtype=0

# Sentinel Shared Key, either the primary or secondary, can be found in Agents Management page under Log Analytics workspace
--sentineloutputsharedkey=

# Sentinel Customer Id. can be found in Agents Management page under Log Analytics workspace
--sentineloutputcustomerid=

# Sentinel Output LogType
--sentineloutputlogtype=dnsmonster

# Sentinel Output Proxy in URI format
--sentineloutputproxy=

# Sentinel Batch Size
--sentinelbatchsize=100

# Interval between sending results to Sentinel if Batch size is not filled. Any value larger than zero takes precedence over Batch Size
--sentinelbatchdelay=0s

# [splunk_output]
# What should be written to HEC. options:
#	0: Disable Output
#	1: Enable Output without any filters
#	2: Enable Output and apply skipdomains logic
#	3: Enable Output and apply allowdomains logic
#	4: Enable Output and apply both skip and allow domains logic
--splunkoutputtype=0

# splunk endpoint address, example: http://127.0.0.1:8088. Used if splunkOutputType is not none, can be specified multiple times for load balanace and HA
--splunkoutputendpoint=

# Splunk HEC Token
--splunkoutputtoken=00000000-0000-0000-0000-000000000000

# Splunk Output Index
--splunkoutputindex=temp

# Splunk Output Proxy in URI format
--splunkoutputproxy=

# Splunk Output Source
--splunkoutputsource=dnsmonster

# Splunk Output Sourcetype
--splunkoutputsourcetype=json

# Send data to HEC in batch sizes
--splunkbatchsize=1000

# Interval between sending results to HEC if Batch size is not filled
--splunkbatchdelay=1s

# [stdout_output]
# What should be written to stdout. options:
#	0: Disable Output
#	1: Enable Output without any filters
#	2: Enable Output and apply skipdomains logic
#	3: Enable Output and apply allowdomains logic
#	4: Enable Output and apply both skip and allow domains logic
--stdoutoutputtype=0

# Output format for stdout. options:json,csv, csv_no_header, gotemplate. note that the csv splits the datetime format into multiple fields
--stdoutoutputformat=json

# Go Template to format the output as needed
--stdoutoutputgotemplate={{.}}

# Number of workers
--stdoutoutputworkercount=8

# [syslog_output]
# What should be written to Syslog server. options:
#	0: Disable Output
#	1: Enable Output without any filters
#	2: Enable Output and apply skipdomains logic
#	3: Enable Output and apply allowdomains logic
#	4: Enable Output and apply both skip and allow domains logic
--syslogoutputtype=0

# Syslog endpoint address, example: udp://127.0.0.1:514, tcp://127.0.0.1:514. Used if syslogOutputType is not none
--syslogoutputendpoint=udp://127.0.0.1:514

# [zinc_output]
# What should be written to zinc. options:
#	0: Disable Output
#	1: Enable Output without any filters
#	2: Enable Output and apply skipdomains logic
#	3: Enable Output and apply allowdomains logic
#	4: Enable Output and apply both skip and allow domains logic
--zincoutputtype=0

# index used to save data in Zinc
--zincoutputindex=dnsmonster

# zinc endpoint address, example: http://127.0.0.1:9200/api/default/default/_bulk. Used if zincOutputType is not none
--zincoutputendpoint=

# zinc username, example: admin@admin.com. Used if zincOutputType is not none
--zincoutputusername=

# zinc password, example: password. Used if zincOutputType is not none
--zincoutputpassword=

# Send data to Zinc in batch sizes
--zincbatchsize=1000

# Interval between sending results to Zinc if Batch size is not filled
--zincbatchdelay=1s

# Zing request timeout
--zinctimeout=10s

# [general]
# Garbage Collection interval for tcp assembly and ip defragmentation
--gctime=10s

# Duration to calculate interface stats
--capturestatsdelay=1s

# Mask IPv4s by bits. 32 means all the bits of IP is saved in DB
--masksize4=32

# Mask IPv6s by bits. 32 means all the bits of IP is saved in DB
--masksize6=128

# Name of the server used to index the metrics.
--servername=default

# Set debug Log format
--logformat=text

# Set debug Log level, 0:PANIC, 1:ERROR, 2:WARN, 3:INFO, 4:DEBUG
--loglevel=3

# Size of the result processor channel size
--resultchannelsize=100000

# write cpu profile to file
--cpuprofile=

# write memory profile to file
--memprofile=

# GOMAXPROCS variable
--gomaxprocs=-1

# Limit of packets logged to clickhouse every iteration. Default 0 (disabled)
--packetlimit=0

# Skip outputing domains matching items in the CSV file path. Can accept a URL (http:// or https://) or path
--skipdomainsfile=

# Hot-Reload skipdomainsfile interval
--skipdomainsrefreshinterval=1m0s

# Allow Domains logic input file. Can accept a URL (http:// or https://) or path
--allowdomainsfile=

# Hot-Reload allowdomainsfile file interval
--allowdomainsrefreshinterval=1m0s

# Skip TLS verification when making HTTPS connections
--skiptlsverification

# [metric]
# Metric Endpoint Service
--metricendpointtype=

# Statsd endpoint. Example: 127.0.0.1:8125 
--metricstatsdagent=

# Prometheus Registry endpoint. Example: http://0.0.0.0:2112/metric
--metricprometheusendpoint=

# Format for  output.
--metricformat=json

# Interval between sending results to Metric Endpoint
--metricflushinterval=10s
```
[//]: <> (end of command line options)

## Environment variables
all the flags can also be set via env variables. Keep in mind that the name of each parameter is always all upper case and the prefix for all the variables is "DNSMONSTER."

Example:

```shell
$ export DNSMONSTER_PORT=53
$ export DNSMONSTER_DEVNAME=lo
$ sudo -E dnsmonster
```

## Configuration file
you can run `dnsmonster` using the following command to use configuration file:

```shell
$ sudo dnsmonster --config=dnsmonster.ini

# Or you can use environment variables to set the configuration file path
$ export DNSMONSTER_CONFIG=dnsmonster.ini
$ sudo -E dnsmonster
```


## What's the retention policy

The default retention policy for the ClickHouse tables is set to 30 days. You can change the number by building the containers using `./autobuild.sh`. Since ClickHouse doesn't have an internal timestamp, the TTL will look at incoming packet's date in `pcap` files. So while importing old `pcap` files, ClickHouse may automatically start removing the data as they're being written and you won't see any actual data in your Grafana. To fix that, you can change TTL to a day older than your earliest packet inside the PCAP file. 

NOTE: to change a TTL at any point in time, you need to directly connect to the Clickhouse server using a `clickhouse` client and run the following SQL statement (this example changes it from 30 to 90 days):
```sql
ALTER TABLE DNS_LOG MODIFY TTL DnsDate + INTERVAL 90 DAY;`
```

NOTE: The above command only changes TTL for the raw DNS log data, which is the majority of your capacity consumption. To make sure that you adjust the TTL for every single aggregation table, you can run the following:

```sql
ALTER TABLE DNS_LOG MODIFY TTL DnsDate + INTERVAL 90 DAY;
ALTER TABLE `.inner.DNS_DOMAIN_COUNT` MODIFY TTL DnsDate + INTERVAL 90 DAY;
ALTER TABLE `.inner.DNS_DOMAIN_UNIQUE` MODIFY TTL DnsDate + INTERVAL 90 DAY;
ALTER TABLE `.inner.DNS_PROTOCOL` MODIFY TTL DnsDate + INTERVAL 90 DAY;
ALTER TABLE `.inner.DNS_GENERAL_AGGREGATIONS` MODIFY TTL DnsDate + INTERVAL 90 DAY;
ALTER TABLE `.inner.DNS_EDNS` MODIFY TTL DnsDate + INTERVAL 90 DAY;
ALTER TABLE `.inner.DNS_OPCODE` MODIFY TTL DnsDate + INTERVAL 90 DAY;
ALTER TABLE `.inner.DNS_TYPE` MODIFY TTL DnsDate + INTERVAL 90 DAY;
ALTER TABLE `.inner.DNS_CLASS` MODIFY TTL DnsDate + INTERVAL 90 DAY;
ALTER TABLE `.inner.DNS_RESPONSECODE` MODIFY TTL DnsDate + INTERVAL 90 DAY;
ALTER TABLE `.inner.DNS_SRCIP_MASK` MODIFY TTL DnsDate + INTERVAL 90 DAY;
```

UPDATE: in the latest version of `clickhouse`, the .inner tables don't have the same name as the corresponding aggregation views. To modify the TTL you have to find the table names in UUID format using `SHOW TABLES` and repeat the `ALTER` command with those UUIDs.

# Sampling and Skipping

## pre-process sampling
`dnsmonster` supports pre-processing sampling of packet using a simple parameter: `sampleRatio`. this parameter accepts a "ratio" value, like `1:2`. `1:2` means for each 2 packet that arrives, only process one of them (50% sampling). Note that this sampling happens AFTER `bpf` filters and not before. if you have an issue keeping up with the volume of your DNS traffic, you can set this to something like `2:10`, meaning 20% of the packets that pass your `bpf` filter, will be processed by `dnsmonster`. 

## skip domains
`dnsmonster` supports a post-processing domain skip list to avoid writing noisy, repetitive data to your Database. The domain skip list is a csv-formatted file, with only two columns: a string and a logic for that particular string. `dnsmonster` supports three logics: `prefix`, `suffix` and `fqdn`. `prefix` and `suffix` means that only the domains starting/ending with the mentioned string will be skipped to be written to DB. Note that since the process is being done on DNS questions, your string will most likely have a trailing `.` that needs to be included in your skip list row as well (take a look at [skipdomains.csv.sample](skipdomains.csv.sample) for a better view). You can also have a full FQDN match to avoid writing highly noisy FQDNs into your database.

## allow domains
`dnsmonster` has the concept of allowdomains, which helps building the detection if certain FQDNs, prefixes or suffixes are present in the DNS traffic. Given the fact that `dnsmonster` supports multiple output streams with different logic for each one, it's possible to collect all DNS traffic in ClickHouse, but collect only allowlist domains in stdout or in a file in the same instance of `dnsmonster`.

## SAMPLE in clickhouse SELECT queries
By default, the main tables created by [tables.sql](clickhouse/tables.sql) (`DNS_LOG`) file have the ability to sample down a result as needed, since each DNS question has a semi-unique UUID associated with it. For more information about SAMPLE queries in Clickhouse, please check out [this](https://clickhouse.tech/docs/en/sql-reference/statements/select/sample/) document.

# Supported Inputs
* Live capture via libpcap/ncap (Ethernet and raw IP are supported)
* Live capture via afpacket (Ethernet and raw IP are supported)
* Dnstap socket (listen mode)
* Pcap file (Ethernet frame)

NOTE: if your `pcap` file is captured by one of Linux's meta-interfaces (for example tcpdump -i any), dnsmonster won't be able to read the Ethernet frame off of it since it doesn't exist. you can use a tool like `tcprewrite` to convert the pcap file to Ethernet.

# Supported Outputs

* Clickhouse
* Kafka
* Elasticsearch
* Splunk HEC
* Stdout
* File
* Syslog (Linux Only)
* Microsoft Sentinel
* InfluxDB


# Roadmap
- [x] Down-sampling capability for SELECT queries
- [x] Adding `afpacket` support
- [x] Configuration file option
- [x] Exclude FQDNs from being indexed
- [x] FQDN whitelisting to only log certain domains
- [x] [dnstap](https://github.com/dnstap/golang-dnstap) support
- [x] Kafka output support
- [x] Ability to load `allowDomains` and `skipDomains` from HTTP(S) endpoints
- [x] Elasticsearch output support
- [x] Splunk HEC output support
- [x] Syslog output support
- [x] Grafana dashboard performance improvements
- [x] remove `libpcap` dependency and move to `pcapgo` for packet processing
- [x] Getting the data ready to be used for ML & Anomaly Detection
- [x] De-duplication support (WIP)
- [x] Optional SSL for Clickhouse
- [x] `statsd` and `Prometheus` support 
- [ ] Splunk Dashboard
- [ ] Kibana Dashbaord
- [ ] Clickhouse versioning and migration tool
- [ ] tests and benchmarks

# Related projects

- [dnszeppelin](https://github.com/niclabs/dnszeppelin)
- [passivedns](https://github.com/gamelinux/passivedns)
- [gopassivedns](https://github.com/Phillipmartin/gopassivedns)
- [packetbeat](https://github.com/elastic/beats/blob/master/packetbeat/)
