---
title: Configuration
description: How dnsmonster is configured — command line flags, environment variables and INI files, and the order they take precedence in.
sidebar:
  label: Overview
  order: 1
---

To run `dnsmonster`, one input and at least one output must be defined. The input can be `devName`
for live packet capture, `pcapFile` to read from a pcap file, or a `dnstapSocket` address to listen
on. Running `dnsmonster` with more than one input stream at a time is not supported.

Output is different: you can have as many output channels as you like, and sometimes multiple
instances of the same output (Splunk, for example) for load balancing and high availability.

When multiple output streams are specified, the output data is copied to all of them. If you set
`stdoutOutputType=1` and `--fileOutputType=1 --fileOutputPath=/dev/stdout`, each processed record
appears twice in your stdout — once from the stdout module, once from the file module that happens
to point at the same place.

## Precedence

`dnsmonster` can be configured three ways, and you can combine them. Highest priority first:

| Source | Case sensitivity | Notes |
| --- | --- | --- |
| Command line options | Case-insensitive | Always wins |
| Environment variables | Always upper case | Prefixed with `DNSMONSTER_` |
| Configuration file | Case-sensitive, lowercase | INI format |
| Default values | — | Used when nothing else is set |

If a configuration file specifies `devName` but you also pass it as a command line argument,
`dnsmonster` prioritises the CLI and ignores that parameter from the `ini` file.

## Command line options

To see the current list of command line options, run `dnsmonster --help`, or check the repository's
`README.md`.

## Environment variables

Every flag can also be set through an environment variable. The name is always upper case and
prefixed with `DNSMONSTER_`:

```shell
export DNSMONSTER_PORT=53
export DNSMONSTER_DEVNAME=lo
sudo -E dnsmonster
```

## Configuration file

Point `dnsmonster` at an INI file with `--config`:

```shell
sudo dnsmonster --config=dnsmonster.ini

# or set the path through the environment
export DNSMONSTER_CONFIG=dnsmonster.ini
sudo -E dnsmonster
```

A fully commented `config-sample.ini` lives in the root of the repository and lists every section
and parameter.
