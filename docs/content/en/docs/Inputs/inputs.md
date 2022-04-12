---
title: "Input options"
linkTitle: "Input options"
weight: 4
---

Let's go through some examples of how to set up `dnsmonster` inputs

### Live interface

To start listening on an interface, simply put the name of the interface in the `--devName=` parameter. In unix-like systems, the `ip a` command or `ifconfig` gives you a list of interfaces that you can use. In this mode, `dnsmonster` needs to run with higher privileges. 

In Windows environments, to get a list of interfaces, open cmd.exe as Administrator and run the following: `getmac.exe`. You'll see a table with your interfaces' MAC address and a Transport Name column with something like this: `\Device\Tcpip_{16000000-0000-0000-0000-145C4638064C}`.

Then you simply replace the word `Tcpip_` with `NPF_` and use it as the `--devName` parameter. Like so

```sh
dnsmonster.exe --devName \Device\NPF_{16000000-0000-0000-0000-145C4638064C}
```
### Pcap file

To analyze a pcap file, you can simply use the `--pcapFile=` option. You can also use the value `-` or `/dev/stdin` to read the `pcap` from stdin. This can be used in pcap-over-ip and zipped pcaps that you would like to analyze on the fly. For example, this example will read the packets as they're getting extracted without saving the extracted pcap on the disk

```sh
lz4cat /path/to/a/hug/dns/capture.pcap.lz4 | dnsmonster --pcapFile=- --stdoutOutputType=1
```

### Pcap-over-Ip

`dnsmonster` doesn't support [pcap-over-ip](https://www.netresec.com/?page=Blog&month=2011-09&post=Pcap-over-IP-in-NetworkMiner) directly, but you can achieve the same results by combining a program like `netcat` or `socat` with `dnsmonster` to make pcap-over-ip work. 

to connect to a remote pcap-over-ip server, use the following

```bash
while true; do
  nc -w 10 REMOTE_IP REMOTE_PORT | dnsmonster --pcapFile=- --stdoutOutputType=1
done
```

to listen on pcap-over-ip, the following code can be used

```bash
while true; do
  nc -l -p REMOTE_PORT | dnsmonster --pcapFile=- --stdoutOutputType=1
done
```

if pcap-over-ip is a popular enough option, the process of building a native capability to support it shouldn't be too difficult. Feel free to open a topic in the discussion page or simply an issue on the repo if this is something you care about. 

### dnstap

`dnsmonster` can listen on a `dnstap` TCP or Unix socket and process the `dnstap` logs as they come in just like a network packet, since `dnstap`'s specification is very close to the packet itself. to learn more about `dnstap`, visit their website [here](https://dnstap.info/). 

to use dnstap as a TCP listener, use `--dnstapSocket` with a syntax like `--dnstapSocket=tcp://0.0.0.0:5555`. If you're using a Unix socket to listen for dnstap packets, you can use `unix:///tmp/dnstap.sock` and set the socket file permission with `--dnstapPermission` option. 

Currently, the `dnstap` in client mode is unsupported since the use case of it is very rare. in case you need this function, you can use a tcp port proxy or `socat` to convert the TCP connection into a unix socket and read it from `dnsmonster`. 