# DNS Monster

Passive DNS collection and monitoring built with Golang, Clickhouse and Grafana: [Blogpost](https://blog.n0p.me/dnsmonster/)

# Configuration

DNSMonster can be configured using 3 different methods. Command line options, Environment variables and configuration file. Order of precedence:

- Command line options
- Environment variables
- Configuration file
- Default values

## Command line options
```shell
Usage of dnsmonster:
  -AfpacketBuffersizeMb=64: Afpacket Buffersize in MB
  -batchSize=100000: Minimun capacity of the cache array used to send data to clickhouse. Set close to the queries per second received to prevent allocations
  -captureStatsDelay=1s: Number of seconds to calculate interface stats
  -clickhouseAddress="localhost:9000": Address of the clickhouse database to save the results
  -clickhouseDebug=false: Debug Clickhouse connection
  -clickhouseDelay=1: Number of seconds to batch the packets
  -clickhouseDryRun=false: process the packets but don't write them to clickhouse. This option will still try to connect to db. For testing only
  -config="": path to config file
  -cpuprofile="": write cpu profile to file
  -defraggerChannelReturnSize=500: Size of the channel where the defragged packets are returned
  -defraggerChannelSize=500: Size of the channel to send packets to be defragged
  -devName="": Device used to capture
  -filter="((ip and (ip[9] == 6 or ip[9] == 17)) or (ip6 and (ip6[6] == 17 or ip6[6] == 6 or ip6[6] == 44)))": BPF filter applied to the packet stream. If port is selected, the packets will not be defragged.
  -gcTime=10: Time in seconds to garbage collect the tcp assembly and ip defragmentation
  -loggerFilename=false: Show the file name and number of the logged string
  -maskSize=32: Mask source IPs by bits. 32 means all the bits of IP is saved in DB
  -memprofile="": write memory profile to file
  -packetHandlerChannelSize=100000: Size of the packet handler channel
  -packetHandlers=1: Number of routines used to handle received packets
  -packetLimit=0: Limit of packets logged to clickhouse every iteration. Default 0 (disabled)
  -pcapFile="": Pcap filename to run
  -port=53: Port selected to filter packets
  -printStatsDelay=10s: Number of seconds to print capture and database stats
  -resultChannelSize=100000: Size of the result processor channel size
  -serverName="default": Name of the server used to index the metrics.
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

# Quick start

## AIO Installation using Docker

![Basic AIO Diagram](static/dnsmonster-basic.svg)

In the example diagram, the egress/ingress of the DNS server traffic is captured, after that, an optional layer of packet aggregation is added before hitting the DNSMonster Server. The outbound data going out of DNS Servers is quite useful to perform cache and performance analysis on the DNS fleet. If an aggregator is not available for you, you can have both TAPs connected directly to DNSMonster and have two DNSMonster Agents looking at the traffic. 

running `./autobuild.sh` creates multiple containers:

* multiple instances of `dnsmonster` to look at the traffic on any interface. Interface list will be prompted as part of `autobuild.sh`
* an instance of `clickhouse` to collect `dnsmonster`'s output and saves all the logs/data to a data and logs directory. Both will be prompted as part of `autobuild.sh`
* an instance of `grafana` looking at the `clickhouse` data with pre-built dashboard.

## What's the retention policy

The default retention policy for the DNS data is set to 30 days. You can change the number by building the containers using `./autobuild.sh`.

NOTE: to change a TTL at any point in time, you need to directly connect to the Clickhouse server using a `clickhouse` client and run the following SQL statement (this example changes it from 30 to 90 days):

`ALTER TABLE DNS_LOG MODIFY TTL DnsDate + INTERVAL 90 DAY;` 

## AIO Demo

[![AIO Demo](static/aio_demo.svg)](static/aio_demo.svg)


## Scalable deployment Howto

### Clickhouse Cluster

![Basic AIO Diagram](static/dnsmonster-enterprise.svg)

### Set up a ClickHouse Cluster

Clickhouse website provides an excellent tutorial on how to create a cluster with a "virtual" table, [reference](https://clickhouse.tech/docs/en/getting-started/tutorial/#cluster-deployment). Note that `DNS_LOG` has to be created virtually in this cluster in order to provide HA and load balancing across the nodes. 

Configuration of Agent as well as Grafana is Coming soon!

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

There are two binary flavours released for each release. A statically-linked self-contained binary built against `musl` on Alpine Linux, which will be maintained [here](`n0p.me/bin/dnsmonster`), and dynamically linked binaries for Windows and Linux, which will depend on `libpcap`. These releases are built against `glibc` so they will have a slight performance advantage over `musl`. These builds will be available in the [release](https://github.com/mosajjal/dnsmonster/releases) section of Github repository. 

## TODO
- [x] Down-sampling capability for SELECT queries
- [x] Adding `afpacket` support
- [x] Configuration file option
- [ ] Splunk Dashboard
- [ ] Exclude FQDNs from being indexed
- [ ] Adding an optional Kafka middleware
- [ ] More DB engine support (Influx, Elasticsearch etc)
- [ ] Getting the data ready to be used for Anomaly Detection
- [ ] Grafana dashboard performance improvements
- [ ] remove libpcap dependency and move to `pcapgo`
