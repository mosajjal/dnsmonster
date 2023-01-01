---
title: "installation"
linkTitle: "installation"
weight: 1
description: >
  Learn how to install dnsmonster on your platform using Docker, prebuilt binaries, or compiling it from the source on any platform Go supports
---

`dnsmonster` has been built with minimum dependencies. In runtime, the only optional dependency for `dnsmonster` is `libpcap`. By building `dnsmonster` without libpcap, you will lose the ability to set `bpf` filters on your live packet captures. 

## installation methods

### Prebuilt binaries

Each relase of `dnsmonster` will ship with two binaries. One for Linux amd64, built statically against an Alpine based image, and one for Windows amd64, which depends on a capture library to be installed on the OS. I've tested thw Windows binary with the latest version of Wireshark installed on the system and there was no issues to run the executable. 

### Prebuilt packages

Per each release, the statically-linked binary mentioned above is also wrapped into `deb` and `rpm` packages with no dependencies, making it easy to deploy it in Debian and RHEL based distributions. Note that the packages don't generate any service files or configuration templates at installation time. 

### Run as a container

The container build process only generates a Linux amd64 output. Since `dnsmonster` uses raw packet capture funcationality, Docker/Podman daemon must grant the capability to the container

```
sudo docker run --rm -it --net=host --cap-add NET_RAW --cap-add NET_ADMIN --name dnsmonster ghcr.io/mosajjal/dnsmonster:latest --devName lo --stdoutOutputType=1
```

Check out the configuration section to understand the provided command line arguments.

### Build from the source

- with `libpcap`:
  Make sure you have `go`, `libpcap-devel` and `linux-headers` packages installed. The name of the packages might differ based on your distribution. After this, simply clone the repository and run `go build .`

```sh
git clone https://github.com/mosajjal/dnsmonster --depth 1 /tmp/dnsmonster 
cd /tmp/dnsmonster
go get
go build -o dnsmonster ./...
```

- without `libpcap`:
`dnsmonster` only uses one function from `libpcap`, and that's converting the `tcpdump`-style filters into BPF bytecode. If you can live with no BPF support, you can build `dnsmonster` without `libpcap`. Note that for any other platform, the packet capture falls back to `libpcap` so it becomes a hard dependency (*BSD, Windows, Darwin)

```sh
git clone https://github.com/mosajjal/dnsmonster --depth 1 /tmp/dnsmonster 
cd /tmp/dnsmonster
go get
go build -o dnsmonster -tags nolibpcap ./...
```

The above build also works on ARMv7 (RPi4) and AArch64.

### Build Statically

If you have a copy of `libpcap.a`, you can build the statically link it to `dnsmonster` and build it fully statically. In the code below, please change `/root/libpcap-1.9.1/libpcap.a` to the location of your copy.

```
git clone https://github.com/mosajjal/dnsmonster --depth 1 /tmp/dnsmonster
cd /tmp/dnsmonster/
go get
go build --ldflags "-L /root/libpcap-1.9.1/libpcap.a -linkmode external -extldflags \"-I/usr/include/libnl3 -lnl-genl-3 -lnl-3 -static\"" -a -o dnsmonster ./...
```

For more information on how the statically linked binary is created, take a look at Dockerfiles in the root of the repository responsible for generating the published binaries.

