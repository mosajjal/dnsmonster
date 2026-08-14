---
title: Input options
description: Worked examples for each dnsmonster input — live interface on Linux and Windows, pcap files, pcap-over-IP and dnstap sockets.
sidebar:
  order: 2
---

Worked examples for each way of getting packets into `dnsmonster`.

## Live interface

To start listening on an interface, put its name in `--devName`. On Unix-like systems, `ip a` or
`ifconfig` lists the interfaces you can use. In this mode `dnsmonster` needs elevated privileges.

On Windows, open `cmd.exe` as Administrator and run `getmac.exe`. You will see a table of MAC
addresses and a Transport Name column with entries like
`\Device\Tcpip_{16000000-0000-0000-0000-145C4638064C}`. Replace `Tcpip_` with `NPF_` and use that as
`--devName`:

```sh
dnsmonster.exe --devName \Device\NPF_{16000000-0000-0000-0000-145C4638064C}
```

## Pcap file

Use `--pcapFile=` to analyse a pcap file. The values `-` and `/dev/stdin` read the pcap from stdin,
which is handy for pcap-over-IP and for compressed captures you want to analyse on the fly. This
example reads packets as they are extracted, without writing the decompressed pcap to disk:

```sh
lz4cat /path/to/a/huge/dns/capture.pcap.lz4 | dnsmonster --pcapFile=- --stdoutOutputType=1
```

## Pcap-over-IP

`dnsmonster` does not support
[pcap-over-IP](https://www.netresec.com/?page=Blog&month=2011-09&post=Pcap-over-IP-in-NetworkMiner)
directly, but you can get the same result by pairing it with `netcat` or `socat`.

To connect to a remote pcap-over-IP server:

```bash
while true; do
  nc -w 10 REMOTE_IP REMOTE_PORT | dnsmonster --pcapFile=- --stdoutOutputType=1
done
```

To listen for pcap-over-IP:

```bash
while true; do
  nc -l -p REMOTE_PORT | dnsmonster --pcapFile=- --stdoutOutputType=1
done
```

If pcap-over-IP turns out to be popular enough, building native support should not be difficult.
Open a discussion topic or an issue on the repo if this matters to you.

## dnstap

`dnsmonster` can listen on a `dnstap` TCP or Unix socket and process dnstap logs as they arrive,
much like network packets — the dnstap specification is close to the packet itself. See
[dnstap.info](https://dnstap.info/) for background.

To listen over TCP:

```sh
dnsmonster --dnstapSocket=tcp://0.0.0.0:5555
```

To listen on a Unix socket, use `unix:///tmp/dnstap.sock` and set the file permission with
`--dnstapPermission`.

`dnstap` in client mode is currently unsupported, since the use case is rare. If you need it, use a
TCP port proxy or `socat` to convert the TCP connection into a Unix socket and read that from
`dnsmonster`.
