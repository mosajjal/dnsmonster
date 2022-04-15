---
title: "Tutorials"
linkTitle: "Tutorials"
weight: 6
date: 2017-01-04
description: >
  Some Design Templates
---

## All-In-One Test Environment

![docker-compose](/img/dnsmonster-autobuild.svg)

Above diagram shows the overview of the autobuild output. running `./autobuild.sh` creates multiple containers:

* a `dnsmonster` container per selected interfaces from the host to look at the raw traffic. Host's interface list will be prompted when running `autobuild.sh`, allowing you to select one or more interfaces.
*a `clickhouse` container to collect `dnsmonster`'s outputs and save all the logs and data to their respective directory inside the host. Both paths will be prompted in `autobuild.sh`. The default tables and TTL for them will implemented automatically.
* a `grafana` container connecting back to `clickhouse`. It automatically sets up the connection to ClickHouse, and sets up the builtin dashboards based on the default ClickHouse tables. Note that Grafana container needs an internet connection to successfully set up the plugins. If you don't have an internet connection, the `dnsmonster` and `clickhouse` containers will start working without any issues, and the error produced by Grafana can be ignored. 

### All-in-one Demo

[![AIO Demo](/img/aio_demo.svg)](/img/aio_demo.svg)
