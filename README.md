Table of Contents
- [DNS Monster](#dns-monster)
- [Main features](#main-features)
- [Manual Installation](#manual-installation)
  - [Linux](#linux)
  - [Windows](#windows)
- [Architecture](#architecture)
  - [AIO Installation using Docker](#aio-installation-using-docker)
    - [AIO Demo](#aio-demo)
  - [Enterprise Deployment](#enterprise-deployment)
    - [Set up a ClickHouse Cluster](#set-up-a-clickhouse-cluster)
- [Configuration](#configuration)
  - [Command line options](#command-line-options)
  - [Environment variables](#environment-variables)
  - [Configuration file](#configuration-file)
  - [What's the retention policy](#whats-the-retention-policy)
- [Sampling and Skipping](#sampling-and-skipping)
  - [pre-process sampling](#pre-process-sampling)
  - [skip domains](#skip-domains)
  - [SAMPLE in clickhouse SELECT queries](#sample-in-clickhouse-select-queries)
- [Build Manually](#build-manually)
  - [Static Build](#static-build)
  - [pre-built Binary](#pre-built-binary)
- [Roadmap](#roadmap)
- [Related projects](#related-projects)

# DNS Monster

Passive DNS collection and monitoring built with Golang, Clickhouse and Grafana: 
`dnsmonster` implements a packet sniffer for DNS traffic. It can accept traffic from a `pcap` file, a live interface or a `dnstap` socket, 
and can be used to index and store thousands of DNS queries per second (it has shown to be capable of indexing 200k+ DNS queries per second on a commodity computer). It aims to be scalable, simple and easy to use, and help
security teams to understand the details about an enterprise's DNS traffic. `dnsmonster` does not look to follow DNS conversations, rather it aims to index DNS packets as soon as they come in. It also does not aim to breach
the privacy of the end-users, with the ability to mask source IP from 1 to 32 bits, making the data potentially untraceable. [Blogpost](https://blog.n0p.me/dnsmonster/)


# Main features

- Can use Linux's `afpacket` and zero-copy packet capture.
- Supports BPF
- Can fuzz source IP to enhance privacy
- Can have a pre-processing sampling ratio
- Can have a list of "skip" `fqdn`s to avoid writing some domains/suffix to storage, thus improving DB performance
- Hot-reload of skip domain file
- Automatic data retention policy using ClickHouse's TTL attribute
- Built-in dashboard using Grafana
- Can be shipped as a single, statically-linked binary
- Ability to be configured using Env variables, command line options or configuration file
- Ability to sample output metrics using ClickHouse's SAMPLE capability
- High compression ratio thanks to ClickHouse's built-in LZ4 storage
- Supports DNS Over TCP, Fragmented DNS (udp/tcp) and IPv6
- Supports [dnstrap](https://github.com/dnstap/golang-dnstap) over Unix socket or TCP

# Manual Installation

## Linux
For `afpacket` v3 support, you need to use kernel 3.x+. Any Linux distro since 5 years ago is shipped with a 3.x+ version so it should work out of the box. The release binary is shipped as a statically-linked binary and shouldn't need any dependencies and will work out of the box. If your distro is not running the pre-compiled version properly, please submit an issue with the details and build `dnsmonster` manually using this section [Build Manually](#build-manually).

## Windows
Windows release of the binary depends on [npcap](https://nmap.org/npcap/#download) to be installed. After installation, the binary should work out of the box. I've tested it in a Windows 10 environment and it ran without an issue. To find interface names to give `-devName` parameter and start sniffing, you'll need to do the following:

  - open cmd.exe (probably as Admin) and run the following: `getmac.exe`, you'll see a table with your interfaces' MAC address and a Transport Name column with something like this: `\Device\Tcpip_{16000000-0000-0000-0000-145C4638064C}`
  - run `dnsmonster.exe` in `cmd.exe` like this:

```batch
dnsmonster.exe \Device\NPF_{16000000-0000-0000-0000-145C4638064C}
```

Note that you should change `\Tcpip` from `getmac.exe` to `\NPF` inside `dnsmonster.exe`.

Since `afpacket` is a Linux feature and Windows is not supported, `useAfpacket` and its related options will not work and will cause unexpected behavior on Windows.

# Architecture

## AIO Installation using Docker

![Basic AIO Diagram](static/dnsmonster-basic.svg)

In the example diagram, the egress/ingress of the DNS server traffic is captured, after that, an optional layer of packet aggregation is added before hitting the DNSMonster Server. The outbound data going out of DNS Servers is quite useful to perform cache and performance analysis on the DNS fleet. If an aggregator is not available for you, you can have both TAPs connected directly to DNSMonster and have two DNSMonster Agents looking at the traffic. 

running `./autobuild.sh` creates multiple containers:

* multiple instances of `dnsmonster` to look at the traffic on any interface. Interface list will be prompted as part of `autobuild.sh`
* an instance of `clickhouse` to collect `dnsmonster`'s output and saves all the logs/data to a data and logs directory. Both will be prompted as part of `autobuild.sh`
* an instance of `grafana` looking at the `clickhouse` data with pre-built dashboard.


### AIO Demo

[![AIO Demo](static/aio_demo.svg)](static/aio_demo.svg)


## Enterprise Deployment


![Basic AIO Diagram](static/dnsmonster-enterprise.svg)

### Set up a ClickHouse Cluster

Clickhouse website provides an excellent tutorial on how to create a cluster with a "virtual" table, [reference](https://clickhouse.tech/docs/en/getting-started/tutorial/#cluster-deployment). Note that `DNS_LOG` has to be created virtually in this cluster in order to provide HA and load balancing across the nodes. 

Configuration of Agent as well as Grafana is Coming soon!

# Configuration

DNSMonster can be configured using 3 different methods. Command line options, Environment variables and configuration file. Order of precedence:

- Command line options
- Environment variables
- Configuration file
- Default values

## Command line options
```
Usage of dnsmonster:
  -AfpacketBuffersizeMb=64: Afpacket Buffersize in MB
  -batchSize=100000: Minimun capacity of the cache array used to send data to clickhouse. Set close to the queries per second received to prevent allocations
  -captureStatsDelay=1s: Duration to calculate interface stats
  -clickhouseAddress="localhost:9000": Address of the clickhouse database to save the results
  -clickhouseDebug=false: Debug Clickhouse connection
  -clickhouseDelay=1s: Interval between sending results to ClickHouse
  -clickhouseDryRun=false: process the packets but don't write them to clickhouse. This option will still try to connect to db. For testing only
  -config="": path to config file
  -cpuprofile="": write cpu profile to file
  -defraggerChannelReturnSize=500: Size of the channel where the defragged packets are returned
  -defraggerChannelSize=500: Size of the channel to send packets to be defragged
  -devName="": Device used to capture
  -dnstapPermission="755": Set the dnstap socket permission, only applicable when unix:// is used
  -dnstapSocket="": dnstrap socket path. Example: unix:///tmp/dnstap.sock, tcp://127.0.0.1:8080
  -filter="((ip and (ip[9] == 6 or ip[9] == 17)) or (ip6 and (ip6[6] == 17 or ip6[6] == 6 or ip6[6] == 44)))": BPF filter applied to the packet stream. If port is selected, the packets will not be defragged.
  -gcTime=10s: Garbage Collection interval for tcp assembly and ip defragmentation
  -gomaxprocs=-1: GOMAXPROCS variable
  -loggerFilename=false: Show the file name and number of the logged string
  -maskSize=32: Mask source IPs by bits. 32 means all the bits of IP is saved in DB
  -memprofile="": write memory profile to file
  -packetHandlerChannelSize=100000: Size of the packet handler channel
  -packetHandlers=1: Number of routines used to handle received packets
  -packetLimit=0: Limit of packets logged to clickhouse every iteration. Default 0 (disabled)
  -pcapFile="": Pcap filename to run
  -port=53: Port selected to filter packets
  -printStatsDelay=10s: Duration to print capture and database stats
  -resultChannelSize=100000: Size of the result processor channel size
  -sampleRatio="1:1": Capture Sampling by a:b. eg sampleRatio of 1:100 will process 1 percent of the incoming packets
  -saveFullQuery=false: Save full packet query and response in JSON format
  -serverName="default": Name of the server used to index the metrics.
  -skipDomainsFile="": Skip saving the domains in the text file, matching the skipDomainsBehavior
  -skipDomainsRefreshInterval=1m0s: Hot-Reload SkipDomains file interval
  -tcpAssemblyChannelSize=1000: Size of the tcp assembler
  -tcpHandlers=1: Number of routines used to handle tcp assembly
  -tcpResultChannelSize=1000: Size of the tcp result channel
  -useAfpacket=false: Use AFPacket for live captures


```


## Environment variables
all the flags can also be set via env variables. Keep in mind that the name of each parameter is always all upper case and the prefix for all the variables is "DNSMONSTER". Example:

```shell
$ export DNSMONSTER_PORT=53
$ export DNSMONSTER_DEVNAME=lo
$ sudo -E dnsmonster
```

## Configuration file
you can run `dnsmonster` using the following command to in order to use configuration file:

```shell
$ sudo dnsmonster -config=dnsmonster.cfg

# Or you can use environment variables to set the configuration file path
$ export DNSMONSTER_CONFIG=dnsmonster.cfg
$ sudo -E dnsmonster
```


## What's the retention policy

The default retention policy for the DNS data is set to 30 days. You can change the number by building the containers using `./autobuild.sh`. Since ClickHouse doesn't have an internal timestamp, the TTL will look at incoming packet's date in `pcap` files. So while importing old `pcap` files, ClickHouse may automatically start removing the data as they're being written and you won't see any actual data in your Grafana. To fix that, you can change TTL to a day older than your earliest packet inside the PCAP file. 

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
ALTER TABLE `.inner.DNS_IP_MASK` MODIFY TTL DnsDate + INTERVAL 90 DAY;
```

# Sampling and Skipping

## pre-process sampling
`dnsmonster` supports pre-processing sampling of packet using a simple parameter: `sampleRatio`. this parameter accepts a "ratio" value, like "1:2". "1:2" means for each 2 packet that arrives, only process one of them (50% sampling). Note that this sampling happens AFTER `bpf` filters and not before. if you have an issue keeping up with the volume of your DNS traffic, you can set this to something like "2:10", meaning 20% of the packets that pass your `bpf` filter, will be processed by `dnsmonster`. 

## skip domains
`dnsmonster` supports a post-processing domain skip list to avoid writing noisy, repetitive data to your Database. The domain skip list is a csv-formatted file, with only two columns: a string and a logic for that particular string. `dnsmonster` supports two logics: `suffix` and `fqdn`. `suffix` means that only the domains ending with the mentioned string will be skipped to be written to DB. Note that since we're talking about DNS questions, your string will most likely have a trailing `.` that needs to be included in your skip list row as well (take a look at [skipdomains.csv.sample](skipdomains.csv.sample) for a better view). You can also have a full FQDN match to avoid writing highly noisy FQDNs into your database.

FQDN prefix will not be supported at this point since the trust tree of a FQDN starts at the end not at the beginning, which will allow a malicious domain to have a prefix of "google.com" in their domain, creating false negatives for `dnsmonster`.

## SAMPLE in clickhouse SELECT queries
By default, the main tables created by [tables.sql](clickhouse/tables.sql) (`DNS_LOG`) file have the ability to sample down a result as needed, since each DNS question has a semi-unique UUID associated with it. For more information about SAMPLE queries in Clickhouse, please check out [this](https://clickhouse.tech/docs/en/sql-reference/statements/select/sample/) document.


# Build Manually

Make sure you have `libpcap-devel` and `linux-headers` packages installed.

`go get gitlab.com/mosajjal/dnsmonster/src`

## Static Build

```
 $ git clone https://gitlab.com/mosajjal/dnsmonster
 $ cd dnsmonster/src/
 $ go get
 $ go build --ldflags "-L /root/libpcap-1.9.1/libpcap.a -linkmode external -extldflags \"-I/usr/include/libnl3 -lnl-genl-3 -lnl-3 -static\"" -a -o dnsmonster
```

For more information on how the statically linked binary is created, take a look at [this](Dockerfile) Dockerfile.

## pre-built Binary

There are two binary flavours released for each release. A statically-linked self-contained binary built against `musl` on Alpine Linux, which will be maintained [here](https://n0p.me/bin/dnsmonster), and dynamically linked binaries for Windows and Linux, which will depend on `libpcap`. These releases are built against `glibc` so they will have a slight performance advantage over `musl`. These builds will be available in the [release](https://github.com/mosajjal/dnsmonster/releases) section of Github repository. 

# Roadmap
- [x] Down-sampling capability for SELECT queries
- [x] Adding `afpacket` support
- [x] Configuration file option
- [ ] Splunk Dashboard
- [x] Exclude FQDNs from being indexed
- [ ] Adding an optional Kafka middleware
- [ ] More DB engine support (Influx, Elasticsearch etc)
- [ ] Getting the data ready to be used for Anomaly Detection
- [ ] Grafana dashboard performance improvements
- [ ] remove libpcap dependency and move to `pcapgo`
- [x] [dnstrap](https://github.com/dnstap/golang-dnstap) support

# Related projects

- [dnszeppelin](https://github.com/niclabs/dnszeppelin)
- [passivedns](https://github.com/gamelinux/passivedns)
- [gopassivedns](https://github.com/Phillipmartin/gopassivedns)
- [packetbeat](https://github.com/elastic/beats/blob/master/packetbeat/)
