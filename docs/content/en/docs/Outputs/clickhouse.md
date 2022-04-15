---
title: "ClickHouse"
linkTitle: "ClickHouse"
weight: 4
---

[ClickHouse](https://clickhouse.com/docs/en/) is a time-series database engine developed by Yandex. It uses a column-oriented design which makes it a good candidate to store hundreds of thousands of DNS queries per second with extremely good compression ratio as well as fast retrieval of data. 

Currently, `dnsmonster`'s implementation requires the table name to be set to DNS_LOG. An SQL schema file is provided by the repository under the `clickhouse` directory. The Grafana dashboard and configuration set provided by `dnsmonster` also corresponds with the ClickHouse schema and can be used to visualize the data.

## configuration parameters

- `--clickhouseAddress`: Address of the ClickHouse database to save the results (default: localhost:9000)
- `--clickhouseUsername`: Username to connect to the ClickHouse database (default: empty)
- `--clickhousePassword`: Password to connect to the ClickHouse database (default: empty)
- `--clickhouseDatabase`: Database to connect to the ClickHouse database (default: default)
- `--clickhouseDelay`: Interval between sending results to ClickHouse (default: 1s) 
- `--clickhouseDebug`: Debug ClickHouse connection  (default: false)
- `--clickhouseCompress`: Compress ClickHouse connection (default: false)
- `--clickhouseSecure`: Use TLS for ClickHouse connection  (default: false)
- `--clickhouseSaveFullQuery`: Save full packet query and response in JSON format. (default: false)
- `--clickhouseOutputType`: ClickHouse output type. Options: (default: 0)
  - 0: Disable Output
  - 1: Enable Output without any filters
  - 2: Enable Output and apply skipdomains logic
  - 3: Enable Output and apply allowdomains logic
  - 4: Enable Output and apply both skip and allow domains logic
- `--clickhouseBatchSize`: Minimum capacity of the cache array used to send data to clickhouse. Set close to the queries per second received to prevent allocations (default: 100000)
- `--clickhouseWorkers`: Number of ClickHouse output Workers (default: 1)
- `--clickhouseWorkerChannelSize`: Channel Size for each ClickHouse Worker (default: 100000)

Note: the general option `--skipTLSVerification` applies to this module as well.

## Retention Policy

The default retention policy for the ClickHouse tables is set to 30 days. You can change the number by building the containers using `./autobuild.sh`. Since ClickHouse doesn't have an internal timestamp, the TTL will look at incoming packet's date in `pcap` files. So while importing old `pcap` files, ClickHouse may automatically start removing the data as they're being written and you won't see any actual data in your Grafana. To fix that, you can change TTL to a day older than your earliest packet inside the PCAP file. 

NOTE: to manually change the TTL, you need to directly connect to the ClickHouse server using the `clickhouse-client` binary and run the following SQL statements (this example changes it from 30 to 90 days):
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

UPDATE: in the latest version of `clickhouse`, the .inner tables don't have the same name as the corresponding aggregation views. To modify the TTL you have to find the table names in UUID format using `SHOW TABLES` and repeat the `ALTER` command with those UUIDs.


## SAMPLE in clickhouse SELECT queries
By default, the main tables created by [tables.sql](clickhouse/tables.sql) (`DNS_LOG`) file have the ability to sample down a result as needed, since each DNS question has a semi-unique UUID associated with it. For more information about SAMPLE queries in Clickhouse, please check out [this](https://clickhouse.tech/docs/en/sql-reference/statements/select/sample/) document.