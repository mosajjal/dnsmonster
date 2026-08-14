---
title: Installation
description: Install dnsmonster with Docker, prebuilt binaries and packages, or compile it from source on any platform Go supports.
sidebar:
  order: 1
---

`dnsmonster` is built with minimum dependencies. At runtime the only optional dependency is
`libpcap`. Building `dnsmonster` without libpcap loses the ability to set `bpf` filters on live
packet captures — everything else keeps working.

## Installation methods

### Prebuilt binaries

Each release of `dnsmonster` ships two binaries: one for Linux amd64, built statically against an
Alpine-based image, and one for Windows amd64, which depends on a capture library being installed on
the OS. The Windows binary has been tested with a current Wireshark install and runs without issues.

### Prebuilt packages

For each release, the statically linked Linux binary is also wrapped into `deb` and `rpm` packages
with no dependencies, which makes it easy to deploy on Debian- and RHEL-based distributions. Note
that the packages do not generate service files or configuration templates at install time — see
[post-installation](/getting-started/post-installation/) for that.

### Run as a container

The container build only produces a Linux amd64 image. Because `dnsmonster` uses raw packet capture,
the Docker/Podman daemon must grant the capability to the container:

```sh
sudo docker run --rm -it --net=host \
  --cap-add NET_RAW --cap-add NET_ADMIN \
  --name dnsmonster ghcr.io/mosajjal/dnsmonster:latest \
  --devName lo --stdoutOutputType=1
```

Check the [configuration section](/configuration/) to understand the command line arguments above.

### Build from source

With `libpcap` — make sure you have `go`, `libpcap-devel` and `linux-headers` installed. Package
names differ between distributions.

```sh
git clone https://github.com/mosajjal/dnsmonster --depth 1 /tmp/dnsmonster
cd /tmp/dnsmonster
go get
go build -o dnsmonster ./cmd/dnsmonster
```

Without `libpcap` — `dnsmonster` uses exactly one function from `libpcap`: converting
`tcpdump`-style filters into BPF bytecode. If you can live without BPF support, build with the
`nolibpcap` tag. On every non-Linux platform, packet capture falls back to `libpcap`, so it becomes a
hard dependency there (\*BSD, Windows, Darwin).

```sh
git clone https://github.com/mosajjal/dnsmonster --depth 1 /tmp/dnsmonster
cd /tmp/dnsmonster
go get
go build -o dnsmonster -tags nolibpcap ./cmd/dnsmonster
```

The above build also works on ARMv7 (Raspberry Pi 4) and AArch64.

### Build statically

If you have a copy of `libpcap.a`, you can link it statically and build a fully static binary.
Change `/root/libpcap-1.9.1/libpcap.a` below to the location of your copy.

```sh
git clone https://github.com/mosajjal/dnsmonster --depth 1 /tmp/dnsmonster
cd /tmp/dnsmonster/
go get
go build --ldflags "-L /root/libpcap-1.9.1/libpcap.a -linkmode external -extldflags \"-I/usr/include/libnl3 -lnl-genl-3 -lnl-3 -static\"" -a -o dnsmonster ./cmd/dnsmonster
```

For more detail on how the statically linked binary is produced, look at the Dockerfiles in the root
of the repository — they generate the published binaries.
