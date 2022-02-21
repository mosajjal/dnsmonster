![Build Status](https://github.com/mosajjal/dnsmonster/workflows/Build%20Test/badge.svg?style=flat-square)
![Go Version](https://img.shields.io/github/go-mod/go-version/mosajjal/dnsmonster/main?filename=go.mod&style=flat-square)
![Latest Version](https://img.shields.io/github/v/tag/mosajjal/dnsmonster?label=latest&style=flat-square)
![License](https://img.shields.io/github/license/mosajjal/dnsmonster?style=flat-square)
![Open Issues](https://img.shields.io/github/issues/mosajjal/dnsmonster?style=flat-square)

![Logo](static/dnsmonster-logo.svg)

Table of Contents
- [DNS Monster](#dns-monster)
- [Main features](#main-features)
- [Installation](#installation)
  - [Linux](#linux)
    - [Container](#container)
    - [Build Manually](#build-manually)
    - [Build Statically](#build-statically)
  - [Windows](#windows)
  - [FreeBSD and MacOS](#freebsd-and-macos)
- [Architecture](#architecture)
  - [AIO Installation using Docker](#aio-installation-using-docker)
    - [AIO Demo](#aio-demo)
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

# DNS Monster

Passive DNS monitoring framework built on Golang. 
`dnsmonster` implements a packet sniffer for DNS traffic. It can accept traffic from a `pcap` file, a live interface or a `dnstap` socket, 
and can be used to index and store hundreds of thousands of DNS queries per second as it has shown to be capable of indexing 200k+ DNS queries per second on a commodity computer. It aims to be scalable, simple and easy to use, and help
security teams to understand the details about an enterprise's DNS traffic. `dnsmonster` does not look to follow DNS conversations, rather it aims to index DNS packets as soon as they come in. It also does not aim to breach
the privacy of the end-users, with the ability to mask Layer 3 IPs (IPv4 and IPv6), enabling teams to perform trend analysis on aggregated data without being able to trace back the queries to an individual. [Blogpost](https://blog.n0p.me/dnsmonster/)

The code before version 1.x is considered beta quality and is subject to breaking changes. Please check the release notes for each tag to see the list of breaking scenarios between each release, and how to mitigate potential data loss.

![Inner logic of dnsmonster](static/dnsmonster-inner.svg)

# Main features

- Can use Linux's `afpacket` and zero-copy packet capture.
- Supports BPF
- Can mask IP to enhance privacy
- Can have a pre-processing sampling ratio
- Can have a list of "skip" `fqdn`s to avoid writing some domains/suffix/prefix to storage, thus improving DB performance
- Can have a list of "allow" domains to only log hits of certain domains in Clickhouse/Stdout/File
- Modular output with different logic per output stream. See [Supported Outputs](#supported-outputs)
- Hot-reload of skip and allow domain files
- Automatic data retention policy using ClickHouse's TTL attribute
- Simple Grafana dashboard for Clickhouse tables
- Can be shipped as a single, statically-linked binary
- Ability to be configured using Env variables, command line options or configuration file
- Ability to sample output metrics using ClickHouse's SAMPLE capability
- High compression ratio thanks to ClickHouse's built-in LZ4 storage
- Supports DNS Over TCP, Fragmented DNS (udp/tcp) and IPv6
- Supports [dnstrap](https://github.com/dnstap/golang-dnstap) over Unix socket or TCP

# Installation

## Linux

Best way to get started with `dnsmonster` is to download the binary from the release section. The binary is statically built against `musl`, hence it should work out of the box for many distros. For `afpacket` support, you must use kernel 3.x+. Any modern Linux distribution (CentOS/RHEL 7+, Ubuntu 14.0.4.2+, Debian 7+) is shipped with a 3.x+ version so it should work out of the box. If your distro is not working with the pre-compiled version properly, please submit an issue with the details, and build `dnsmonster` manually using this section [Build Manually](#build-manually).

### Container

Since `dnsmonster` uses raw packet capture funcationality, Docker/Podman daemon must grant the capability to the container

```
sudo docker run --rm -it --net=host --cap-add NET_RAW --cap-add NET_ADMIN --name dnsmonster ghcr.io/mosajjal/dnsmonster:latest --devName lo --stdoutOutputType=1
```


### Build Manually

- with `libpcap`:
  Make sure you have `go`, `libpcap-devel` and `linux-headers` packages installed. The name of the packages might differ based on your distribution. After this, simply clone the repository and run `go build .`

```sh
git clone https://github.com/mosajjal/dnsmonster --depth 1 /tmp/dnsmonster 
cd /tmp/dnsmonster
go get
go build -o dnsmonster .
```

- without `libpcap`:
`dnsmonster` only uses one function from `libpcap`, and that is converting the `tcpdump`-style filters into BPF bytecode. If you can live with no BPF support, you can build `dnsmonster` without `libpcap`. Note that for any other platform, the packet capture falls back to `libpcap` so it becomes a hard dependency (*BSD, Windows, Darwin)

```sh
git clone https://github.com/mosajjal/dnsmonster --depth 1 /tmp/dnsmonster 
cd /tmp/dnsmonster
go get
go build -o dnsmonster -tags nolibpcap .
```

The above build also works on ARMv7 (RPi4) and AArch64.

### Build Statically

If you have a copy of `libpcap.a`, you can build the statically link it to `dnsmonster` and build it fully statically. In the code below, please change `/root/libpcap-1.9.1/libpcap.a` to the location of your copy.

```
git clone https://github.com/mosajjal/dnsmonster --depth 1 /tmp/dnsmonster
cd /tmp/dnsmonster/
go get
go build --ldflags "-L /root/libpcap-1.9.1/libpcap.a -linkmode external -extldflags \"-I/usr/include/libnl3 -lnl-genl-3 -lnl-3 -static\"" -a -o dnsmonster
```

For more information on how the statically linked binary is created, take a look at [this](Dockerfile) Dockerfile.

## Windows

Bulding on Windows is much the same as Linux. Just make sure that you have `npcap`. Clone the repository (`--history 1` works), and run `go get` and `go build .`

As mentioned, Windows release of the binary depends on [npcap](https://nmap.org/npcap/#download) to be installed. After installation, the binary should work out of the box. I've tested it in a Windows 10 environment and it ran without an issue. To find interface names to give `-devName` parameter and start sniffing, you'll need to do the following:

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
go build -o dnsmonster .
```

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


# Configuration

DNSMonster can be configured using 3 different methods. Command line options, Environment variables and configuration file. Order of precedence:

- Command line options (Case-sensitive, camelCase)
- Environment variables (Always upper-case)
- Configuration file (Case-sensitive, PascalCase)
- Default values (No configuration)

## Command line options
[//]: <> (start of command line options)
```
  dnsmonster

capture:
      --devName=                         Device used to capture
                                         [$DNSMONSTER_DEVNAME]
      --pcapFile=                        Pcap filename to run
                                         [$DNSMONSTER_PCAPFILE]
      --dnstapSocket=                    dnstrap socket path. Example:
                                         unix:///tmp/dnstap.sock,
                                         tcp://127.0.0.1:8080
                                         [$DNSMONSTER_DNSTAPSOCKET]
      --port=                            Port selected to filter packets
                                         (default: 53) [$DNSMONSTER_PORT]
      --sampleRatio=                     Capture Sampling by a:b. eg
                                         sampleRatio of 1:100 will process 1
                                         percent of the incoming packets
                                         (default: 1:1)
                                         [$DNSMONSTER_SAMPLERATIO]
      --dedupCleanupInterval=            Cleans up packet hash table used for
                                         deduplication (default: 60s)
                                         [$DNSMONSTER_DEDUPCLEANUPINTERVAL]
      --dnstapPermission=                Set the dnstap socket permission, only
                                         applicable when unix:// is used
                                         (default: 755)
                                         [$DNSMONSTER_DNSTAPPERMISSION]
      --packetHandlerCount=              Number of routines used to handle
                                         received packets (default: 2)
                                         [$DNSMONSTER_PACKETHANDLERCOUNT]
      --tcpAssemblyChannelSize=          Size of the tcp assembler (default:
                                         10000)
                                         [$DNSMONSTER_TCPASSEMBLYCHANNELSIZE]
      --tcpResultChannelSize=            Size of the tcp result channel
                                         (default: 10000)
                                         [$DNSMONSTER_TCPRESULTCHANNELSIZE]
      --tcpHandlerCount=                 Number of routines used to handle tcp
                                         assembly (default: 1)
                                         [$DNSMONSTER_TCPHANDLERCOUNT]
      --defraggerChannelSize=            Size of the channel to send packets to
                                         be defragged (default: 10000)
                                         [$DNSMONSTER_DEFRAGGERCHANNELSIZE]
      --defraggerChannelReturnSize=      Size of the channel where the
                                         defragged packets are returned
                                         (default: 10000)
                                         [$DNSMONSTER_DEFRAGGERCHANNELRETURNSIZ-

                                         E]
      --packetChannelSize=               Size of the packet handler channel
                                         (default: 1000)
                                         [$DNSMONSTER_PACKETCHANNELSIZE]
      --afpacketBuffersizeMb=            Afpacket Buffersize in MB (default:
                                         64) [$DNSMONSTER_AFPACKETBUFFERSIZEMB]
      --filter=                          BPF filter applied to the packet
                                         stream. If port is selected, the
                                         packets will not be defragged.
                                         (default: ((ip and (ip[9] == 6 or
                                         ip[9] == 17)) or (ip6 and (ip6[6] ==
                                         17 or ip6[6] == 6 or ip6[6] == 44))))
                                         [$DNSMONSTER_FILTER]
      --useAfpacket                      Use AFPacket for live captures.
                                         Supported on Linux 3.0+ only
                                         [$DNSMONSTER_USEAFPACKET]
      --noEtherframe                     The PCAP capture does not contain
                                         ethernet frames
                                         [$DNSMONSTER_NOETHERFRAME]
      --dedup                            Deduplicate incoming packets, Only
                                         supported with --devName and
                                         --pcapFile. Experimental
                                         [$DNSMONSTER_DEDUP]

clickhouse_output:
      --clickhouseAddress=               Address of the clickhouse database to
                                         save the results (default:
                                         localhost:9000)
                                         [$DNSMONSTER_CLICKHOUSEADDRESS]
      --clickhouseUsername=              Username to connect to the clickhouse
                                         database
                                         [$DNSMONSTER_CLICKHOUSEUSERNAME]
      --clickhousePassword=              Password to connect to the clickhouse
                                         database
                                         [$DNSMONSTER_CLICKHOUSEPASSWORD]
      --clickhouseDatabase=              Database to connect to the clickhouse
                                         database (default: default)
                                         [$DNSMONSTER_CLICKHOUSEDATABASE]
      --clickhouseDelay=                 Interval between sending results to
                                         ClickHouse (default: 1s)
                                         [$DNSMONSTER_CLICKHOUSEDELAY]
      --clickhouseDebug                  Debug Clickhouse connection
                                         [$DNSMONSTER_CLICKHOUSEDEBUG]
      --clickhouseCompress               Compress Clickhouse connection
                                         [$DNSMONSTER_CLICKHOUSECOMPRESS]
      --clickhouseSecure                 Use TLS for Clickhouse connection
                                         [$DNSMONSTER_CLICKHOUSESECURE]
      --clickhouseSaveFullQuery          Save full packet query and response in
                                         JSON format.
                                         [$DNSMONSTER_CLICKHOUSESAVEFULLQUERY]
      --clickhouseOutputType=[0|1|2|3|4] What should be written to clickhouse.
                                         options:
                                         ;	0: Disable Output
                                         ;	1: Enable Output without any filters
                                         ;	2: Enable Output and apply
                                         skipdomains logic
                                         ;	3: Enable Output and apply
                                         allowdomains logic
                                         ;	4: Enable Output and apply both skip
                                         and allow domains logic (default: 0)
                                         [$DNSMONSTER_CLICKHOUSEOUTPUTTYPE]
      --clickhouseBatchSize=             Minimun capacity of the cache array
                                         used to send data to clickhouse. Set
                                         close to the queries per second
                                         received to prevent allocations
                                         (default: 100000)
                                         [$DNSMONSTER_CLICKHOUSEBATCHSIZE]
      --clickhouseWorkers=               Number of Clickhouse output Workers
                                         (default: 1)
                                         [$DNSMONSTER_CLICKHOUSEWORKERS]
      --clickhouseWorkerChannelSize=     Channel Size for each Clickhouse
                                         Worker (default: 100000)
                                         [$DNSMONSTER_CLICKHOUSEWORKERCHANNELSI-

                                         ZE]

elastic_output:
      --elasticOutputType=[0|1|2|3|4]    What should be written to elastic.
                                         options:
                                         ;	0: Disable Output
                                         ;	1: Enable Output without any filters
                                         ;	2: Enable Output and apply
                                         skipdomains logic
                                         ;	3: Enable Output and apply
                                         allowdomains logic
                                         ;	4: Enable Output and apply both skip
                                         and allow domains logic (default: 0)
                                         [$DNSMONSTER_ELASTICOUTPUTTYPE]
      --elasticOutputEndpoint=           elastic endpoint address, example:
                                         http://127.0.0.1:9200. Used if
                                         elasticOutputType is not none
                                         [$DNSMONSTER_ELASTICOUTPUTENDPOINT]
      --elasticOutputIndex=              elastic index (default: default)
                                         [$DNSMONSTER_ELASTICOUTPUTINDEX]
      --elasticBatchSize=                Send data to Elastic in batch sizes
                                         (default: 1000)
                                         [$DNSMONSTER_ELASTICBATCHSIZE]
      --elasticBatchDelay=               Interval between sending results to
                                         Elastic if Batch size is not filled
                                         (default: 1s)
                                         [$DNSMONSTER_ELASTICBATCHDELAY]

file_output:
      --fileOutputType=[0|1|2|3|4]       What should be written to file.
                                         options:
                                         ;	0: Disable Output
                                         ;	1: Enable Output without any filters
                                         ;	2: Enable Output and apply
                                         skipdomains logic
                                         ;	3: Enable Output and apply
                                         allowdomains logic
                                         ;	4: Enable Output and apply both skip
                                         and allow domains logic (default: 0)
                                         [$DNSMONSTER_FILEOUTPUTTYPE]
      --fileOutputPath=                  Path to output file. Used if
                                         fileOutputType is not none
                                         [$DNSMONSTER_FILEOUTPUTPATH]
      --fileOutputFormat=[json|csv]      Output format for file.
                                         options:json,csv. note that the csv
                                         splits the datetime format into
                                         multiple fields (default: json)
                                         [$DNSMONSTER_FILEOUTPUTFORMAT]

influx_output:
      --influxOutputType=[0|1|2|3|4]     What should be written to influx.
                                         options:
                                         ;	0: Disable Output
                                         ;	1: Enable Output without any filters
                                         ;	2: Enable Output and apply
                                         skipdomains logic
                                         ;	3: Enable Output and apply
                                         allowdomains logic
                                         ;	4: Enable Output and apply both skip
                                         and allow domains logic (default: 0)
                                         [$DNSMONSTER_INFLUXOUTPUTTYPE]
      --influxOutputServer=              influx Server address, example:
                                         http://localhost:8086. Used if
                                         influxOutputType is not none
                                         [$DNSMONSTER_INFLUXOUTPUTSERVER]
      --influxOutputToken=               Influx Server Auth Token (default:
                                         dnsmonster)
                                         [$DNSMONSTER_INFLUXOUTPUTTOKEN]
      --influxOutputBucket=              Influx Server Bucket (default:
                                         dnsmonster)
                                         [$DNSMONSTER_INFLUXOUTPUTBUCKET]
      --influxOutputOrg=                 Influx Server Org (default:
                                         dnsmonster)
                                         [$DNSMONSTER_INFLUXOUTPUTORG]
      --influxOutputWorkers=             Minimun capacity of the cache array
                                         used to send data to Influx (default:
                                         8) [$DNSMONSTER_INFLUXOUTPUTWORKERS]
      --influxBatchSize=                 Minimun capacity of the cache array
                                         used to send data to Influx (default:
                                         1000) [$DNSMONSTER_INFLUXBATCHSIZE]

kafka_output:
      --kafkaOutputType=[0|1|2|3|4]      What should be written to kafka.
                                         options:
                                         ;	0: Disable Output
                                         ;	1: Enable Output without any filters
                                         ;	2: Enable Output and apply
                                         skipdomains logic
                                         ;	3: Enable Output and apply
                                         allowdomains logic
                                         ;	4: Enable Output and apply both skip
                                         and allow domains logic (default: 0)
                                         [$DNSMONSTER_KAFKAOUTPUTTYPE]
      --kafkaOutputBroker=               kafka broker address(es), example:
                                         127.0.0.1:9092. Used if
                                         kafkaOutputType is not none
                                         [$DNSMONSTER_KAFKAOUTPUTBROKER]
      --kafkaOutputTopic=                Kafka topic for logging (default:
                                         dnsmonster)
                                         [$DNSMONSTER_KAFKAOUTPUTTOPIC]
      --kafkaBatchSize=                  Minimun capacity of the cache array
                                         used to send data to Kafka (default:
                                         1000) [$DNSMONSTER_KAFKABATCHSIZE]
      --kafkaTimeout=                    Kafka connection timeout in seconds
                                         (default: 3) [$DNSMONSTER_KAFKATIMEOUT]
      --kafkaBatchDelay=                 Interval between sending results to
                                         Kafka if Batch size is not filled
                                         (default: 1s)
                                         [$DNSMONSTER_KAFKABATCHDELAY]
      --kafkaCompress                    Compress Kafka connection
                                         [$DNSMONSTER_KAFKACOMPRESS]
      --kafkaSecure                      Use TLS for kafka connection
                                         [$DNSMONSTER_KAFKASECURE]
      --kafkaCACertificatePath=          Path of CA certificate that signs
                                         Kafka broker certificate
                                         [$DNSMONSTER_KAFKACACERTIFICATEPATH]
      --kafkaTLSCertificatePath=         Path of TLS certificate to present to
                                         broker
                                         [$DNSMONSTER_KAFKATLSCERTIFICATEPATH]
      --kafkaTLSKeyPath=                 Path of TLS certificate key
                                         [$DNSMONSTER_KAFKATLSKEYPATH]

sentinel_output:
      --sentinelOutputType=[0|1|2|3|4]   What should be written to Microsoft
                                         Sentinel. options:
                                         ;	0: Disable Output
                                         ;	1: Enable Output without any filters
                                         ;	2: Enable Output and apply
                                         skipdomains logic
                                         ;	3: Enable Output and apply
                                         allowdomains logic
                                         ;	4: Enable Output and apply both skip
                                         and allow domains logic (default: 0)
                                         [$DNSMONSTER_SENTINELOUTPUTTYPE]
      --sentinelOutputSharedKey=         Sentinel Shared Key, either the
                                         primary or secondary, can be found in
                                         Agents Management page under Log
                                         Analytics workspace
                                         [$DNSMONSTER_SENTINELOUTPUTSHAREDKEY]
      --sentinelOutputCustomerId=        Sentinel Customer Id. can be found in
                                         Agents Management page under Log
                                         Analytics workspace
                                         [$DNSMONSTER_SENTINELOUTPUTCUSTOMERID]
      --sentinelOutputLogType=           Sentinel Output LogType (default:
                                         dnsmonster)
                                         [$DNSMONSTER_SENTINELOUTPUTLOGTYPE]
      --sentinelOutputProxy=             Sentinel Output Proxy in URI format
                                         [$DNSMONSTER_SENTINELOUTPUTPROXY]
      --sentinelBatchSize=               Sentinel Batch Size (default: 100)
                                         [$DNSMONSTER_SENTINELBATCHSIZE]
      --sentinelBatchDelay=              Interval between sending results to
                                         Sentinel if Batch size is not filled
                                         (default: 1s)
                                         [$DNSMONSTER_SENTINELBATCHDELAY]

splunk_output:
      --splunkOutputType=[0|1|2|3|4]     What should be written to HEC. options:
                                         ;	0: Disable Output
                                         ;	1: Enable Output without any filters
                                         ;	2: Enable Output and apply
                                         skipdomains logic
                                         ;	3: Enable Output and apply
                                         allowdomains logic
                                         ;	4: Enable Output and apply both skip
                                         and allow domains logic (default: 0)
                                         [$DNSMONSTER_SPLUNKOUTPUTTYPE]
      --splunkOutputEndpoint=            splunk endpoint address, example:
                                         http://127.0.0.1:8088. Used if
                                         splunkOutputType is not none, can be
                                         specified multiple times for load
                                         balanace and HA
                                         [$DNSMONSTER_SPLUNKOUTPUTENDPOINT]
      --splunkOutputToken=               Splunk HEC Token (default:
                                         00000000-0000-0000-0000-000000000000)
                                         [$DNSMONSTER_SPLUNKOUTPUTTOKEN]
      --splunkOutputIndex=               Splunk Output Index (default: temp)
                                         [$DNSMONSTER_SPLUNKOUTPUTINDEX]
      --splunkOutputSource=              Splunk Output Source (default:
                                         dnsmonster)
                                         [$DNSMONSTER_SPLUNKOUTPUTSOURCE]
      --splunkOutputSourceType=          Splunk Output Sourcetype (default:
                                         json)
                                         [$DNSMONSTER_SPLUNKOUTPUTSOURCETYPE]
      --splunkBatchSize=                 Send data to HEC in batch sizes
                                         (default: 1000)
                                         [$DNSMONSTER_SPLUNKBATCHSIZE]
      --splunkBatchDelay=                Interval between sending results to
                                         HEC if Batch size is not filled
                                         (default: 1s)
                                         [$DNSMONSTER_SPLUNKBATCHDELAY]

stdout_output:
      --stdoutOutputType=[0|1|2|3|4]     What should be written to stdout.
                                         options:
                                         ;	0: Disable Output
                                         ;	1: Enable Output without any filters
                                         ;	2: Enable Output and apply
                                         skipdomains logic
                                         ;	3: Enable Output and apply
                                         allowdomains logic
                                         ;	4: Enable Output and apply both skip
                                         and allow domains logic (default: 0)
                                         [$DNSMONSTER_STDOUTOUTPUTTYPE]
      --stdoutOutputFormat=[json|csv]    Output format for stdout.
                                         options:json,csv. note that the csv
                                         splits the datetime format into
                                         multiple fields (default: json)
                                         [$DNSMONSTER_STDOUTOUTPUTFORMAT]
      --stdoutOutputWorkerCount=         Number of workers (default: 8)
                                         [$DNSMONSTER_STDOUTOUTPUTWORKERCOUNT]

syslog_output:
      --syslogOutputType=[0|1|2|3|4]     What should be written to Syslog
                                         server. options:
                                         ;	0: Disable Output
                                         ;	1: Enable Output without any filters
                                         ;	2: Enable Output and apply
                                         skipdomains logic
                                         ;	3: Enable Output and apply
                                         allowdomains logic
                                         ;	4: Enable Output and apply both skip
                                         and allow domains logic (default: 0)
                                         [$DNSMONSTER_SYSLOGOUTPUTTYPE]
      --syslogOutputEndpoint=            Syslog endpoint address, example:
                                         udp://127.0.0.1:514,
                                         tcp://127.0.0.1:514. Used if
                                         syslogOutputType is not none (default:
                                         udp://127.0.0.1:514)
                                         [$DNSMONSTER_SYSLOGOUTPUTENDPOINT]

general:
      --config=                          path to config file
                                         [$DNSMONSTER_CONFIG]
      --gcTime=                          Garbage Collection interval for tcp
                                         assembly and ip defragmentation
                                         (default: 10s) [$DNSMONSTER_GCTIME]
      --captureStatsDelay=               Duration to calculate interface stats
                                         (default: 1s)
                                         [$DNSMONSTER_CAPTURESTATSDELAY]
      --printStatsDelay=                 Duration to print capture and database
                                         stats (default: 10s)
                                         [$DNSMONSTER_PRINTSTATSDELAY]
      --maskSize4=                       Mask IPv4s by bits. 32 means all the
                                         bits of IP is saved in DB (default:
                                         32) [$DNSMONSTER_MASKSIZE4]
      --maskSize6=                       Mask IPv6s by bits. 32 means all the
                                         bits of IP is saved in DB (default:
                                         128) [$DNSMONSTER_MASKSIZE6]
      --serverName=                      Name of the server used to index the
                                         metrics. (default: default)
                                         [$DNSMONSTER_SERVERNAME]
      --logLevel=[0|1|2|3|4]             Set debug Log level, 0:PANIC, 1:ERROR,
                                         2:WARN, 3:INFO, 4:DEBUG (default: 3)
                                         [$DNSMONSTER_LOGLEVEL]
      --resultChannelSize=               Size of the result processor channel
                                         size (default: 100000)
                                         [$DNSMONSTER_RESULTCHANNELSIZE]
      --cpuprofile=                      write cpu profile to file
                                         [$DNSMONSTER_CPUPROFILE]
      --memprofile=                      write memory profile to file
                                         [$DNSMONSTER_MEMPROFILE]
      --gomaxprocs=                      GOMAXPROCS variable (default: -1)
                                         [$DNSMONSTER_GOMAXPROCS]
      --packetLimit=                     Limit of packets logged to clickhouse
                                         every iteration. Default 0 (disabled)
                                         (default: 0) [$DNSMONSTER_PACKETLIMIT]
      --skipDomainsFile=                 Skip outputing domains matching items
                                         in the CSV file path. Can accept a URL
                                         (http:// or https://) or path
                                         [$DNSMONSTER_SKIPDOMAINSFILE]
      --skipDomainsRefreshInterval=      Hot-Reload skipDomainsFile interval
                                         (default: 60s)
                                         [$DNSMONSTER_SKIPDOMAINSREFRESHINTERVA-

                                         L]
      --skipDomainsFileType=             skipDomainsFile type. Options: csv and
                                         hashtable. Hashtable is ONLY fqdn, csv
                                         can support fqdn, prefix and suffix
                                         logic but it's much slower (default:
                                         csv) [$DNSMONSTER_SKIPDOMAINSFILETYPE]
      --allowDomainsFile=                Allow Domains logic input file. Can
                                         accept a URL (http:// or https://) or
                                         path [$DNSMONSTER_ALLOWDOMAINSFILE]
      --allowDomainsRefreshInterval=     Hot-Reload allowDomainsFile file
                                         interval (default: 60s)
                                         [$DNSMONSTER_ALLOWDOMAINSREFRESHINTERV-

                                         AL]
      --allowDomainsFileType=            allowDomainsFile type. Options: csv
                                         and hashtable. Hashtable is ONLY fqdn,
                                         csv can support fqdn, prefix and
                                         suffix logic but it's much slower
                                         (default: csv)
                                         [$DNSMONSTER_ALLOWDOMAINSFILETYPE]
      --skipTLSVerification              Skip TLS verification when making
                                         HTTPS connections
                                         [$DNSMONSTER_SKIPTLSVERIFICATION]
      --version                          show version and quit.
                                         [$DNSMONSTER_VERSION]

help:
  -h, --help                             Print this help to stdout
      --manPage                          Print Manpage for dnsmonster to stdout
      --bashCompletion                   Print bash completion script to stdout
      --fishCompletion                   Print fish completion script to stdout
      --writeConfig=                     generate a config file based on
                                         current inputs (flags, input config
                                         file and environment variables) and
                                         write to provided path
```
[//]: <> (end of command line options)

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
$ sudo dnsmonster -config=dnsmonster.ini

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

UPDATE: in the latest version of `clickhouse`, the .inner tables do not have the same name as the corresponding aggregation views. In order to modify the TTL you have to find the table names in UUID format using `SHOW TABLES` and repeat the `ALTER` command with those UUIDs.

# Sampling and Skipping

## pre-process sampling
`dnsmonster` supports pre-processing sampling of packet using a simple parameter: `sampleRatio`. this parameter accepts a "ratio" value, like "1:2". "1:2" means for each 2 packet that arrives, only process one of them (50% sampling). Note that this sampling happens AFTER `bpf` filters and not before. if you have an issue keeping up with the volume of your DNS traffic, you can set this to something like "2:10", meaning 20% of the packets that pass your `bpf` filter, will be processed by `dnsmonster`. 

## skip domains
`dnsmonster` supports a post-processing domain skip list to avoid writing noisy, repetitive data to your Database. The domain skip list is a csv-formatted file, with only two columns: a string and a logic for that particular string. `dnsmonster` supports three logics: `prefix`, `suffix` and `fqdn`. `prefix` and `suffix` means that only the domains starting/ending with the mentioned string will be skipped to be written to DB. Note that since we're talking about DNS questions, your string will most likely have a trailing `.` that needs to be included in your skip list row as well (take a look at [skipdomains.csv.sample](skipdomains.csv.sample) for a better view). You can also have a full FQDN match to avoid writing highly noisy FQDNs into your database.

## allow domains
`dnsmonster` has the concept of "allowdomains", which helps building the detection if certain FQDNs, prefixes or suffixes are present in the DNS traffic. Given the fact that `dnsmonster` supports multiple output streams with different logic for each one, it's possible to collect all DNS traffic in ClickHouse, but collect only "allowlist" domains in stdout or in a file in the same instance of `dnsmonster`.

## SAMPLE in clickhouse SELECT queries
By default, the main tables created by [tables.sql](clickhouse/tables.sql) (`DNS_LOG`) file have the ability to sample down a result as needed, since each DNS question has a semi-unique UUID associated with it. For more information about SAMPLE queries in Clickhouse, please check out [this](https://clickhouse.tech/docs/en/sql-reference/statements/select/sample/) document.

# Supported Inputs
* Live capture via libpcap/ncap (Ethernet and raw IP are supported)
* Live capture via afpacket (Ethernet and raw IP are supported)
* Dnstap socket (listen mode)
* Pcap file (Ethernet frame)

NOTE: if your `pcap` file is captured by one of Linux's meta-interfaces (eg tcpdump -i any), dnsmonster will not be able to read the Ethernet frame off of it since it doesn't exist. you can use a tool like `tcprewrite` to convert the pcap file to Ethernet.

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
- [ ] Splunk Dashboard
- [ ] Kibana Dashbaord
- [ ] Clickhouse versioning and migration tool
- [ ] `statsd` and `Prometheus` support 
- [ ] tests and benchmarks

# Related projects

- [dnszeppelin](https://github.com/niclabs/dnszeppelin)
- [passivedns](https://github.com/gamelinux/passivedns)
- [gopassivedns](https://github.com/Phillipmartin/gopassivedns)
- [packetbeat](https://github.com/elastic/beats/blob/master/packetbeat/)
