+++
title = "Tutorials"
description = "End-to-end walkthroughs — the all-in-one test environment, ClickHouse Cloud and Kubernetes."
weight = 5
+++

## All-in-one test environment

![Overview of the containers created by autobuild.sh](/img/dnsmonster-autobuild.svg)

Running `./autobuild.sh` from the repository root creates several containers:

- A `dnsmonster` container per selected host interface, looking at raw traffic. The host's interface
  list is prompted when running `autobuild.sh`, so you can select one or more.
- A `clickhouse` container collecting `dnsmonster`'s output and saving logs and data to their
  respective directories on the host. Both paths are prompted by `autobuild.sh`. The default tables
  and their TTL are created automatically.
- A `grafana` container connected back to `clickhouse`. It sets up the ClickHouse connection and the
  built-in dashboards for the default tables. Grafana needs an internet connection to install its
  plugins — without one, `dnsmonster` and `clickhouse` still work fine and the Grafana error can be
  ignored.

### Demo

[![All-in-one demo](/img/aio_demo.svg)](/img/aio_demo.svg)

## Next

- [ClickHouse Cloud](/tutorials/clickhouse-cloud/) — serverless ClickHouse, tables, dictionaries and Grafana.
- [Kubernetes](/tutorials/kubernetes/) — a dnstap logger in `coredns` feeding a `dnsmonster` pod.
