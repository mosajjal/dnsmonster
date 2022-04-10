---
title: "Configuration"
linkTitle: "Configuration"
weight: 2
description: >
  Learn about the command line arguments and what they mean
---

to run `dnsmonster`, one input and at least one output must be defined. The input could be any of `devName` for live packet capture, `pcapFile` to read off a pcap file, or `dnstapSocket` address to listen to. Currently, running `dnsmonster` with more than one input stream at a time is not supported. For output however, it's supported to have more than one channel. In some cases, it's also possible to have multiple instances of the same output (eg Splunk) to provide load balancing and high availability.

Note that in case of specifying different output streams, the output data is replicated across all. For example, if you put `stdoutOutputType=1` and `--fileOutputType=1 --fileOutputPath=/dev/stdout`, you'll see each packet twice in your stdout. One coming from the stdout output type, and the other from the file output type which happens to have the same address (`/dev/stdout`).  

dnsmonster can be configured in 3 different ways. Command line options, Environment variables and a configuration file. You can also use any combination of them at the same time. The precedence order is as follows:

- Command line options (Case-sensitive, camelCase)
- Environment variables (Always upper-case)
- Configuration file (Case-sensitive, PascalCase)
- Default values (No configuration)

For example, if you have a configuration file that has specified a `devName`, but you also provide it as a command-line argument, dnsmonster will prioritizes CLI over config file and will ignore that parameter from the `ini` file. 

## Command line options

Note that command line arguments are case-sensitive and camelCase at the moment. This is the [known limitation](https://github.com/jessevdk/go-flags/issues/333) of the underlying flag parser library `dnsmonster` uses. 

To see the current list of command-line options, run `dnsmonster --help` or checkout the repository's README.md.

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
$ sudo dnsmonster --config=dnsmonster.ini

# Or you can use environment variables to set the configuration file path
$ export DNSMONSTER_CONFIG=dnsmonster.ini
$ sudo -E dnsmonster
```
