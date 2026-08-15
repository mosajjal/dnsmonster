+++
title = "ClickHouse"
description = "Configure the dnsmonster ClickHouse output, set retention with TTL, use SAMPLE queries and run the bundled Grafana dashboard."
weight = 2
+++

[ClickHouse](https://clickhouse.com/docs/en/) is a column-oriented time series database engine. That
design makes it a strong candidate for storing hundreds of thousands of DNS queries per second, with
a very good compression ratio and fast retrieval.

`dnsmonster`'s implementation currently requires the table name to be `DNS_LOG`. An SQL schema file
is provided in the repository under the `clickhouse` directory. The Grafana dashboard and
configuration set shipped with `dnsmonster` matches that schema and can be used to visualise the
data directly.

## Configuration parameters

| Flag | Default | Purpose |
| --- | --- | --- |
| `--clickhouseAddress` | `localhost:9000` | Address of the ClickHouse database. |
| `--clickhouseUsername` | empty | Username for the connection. |
| `--clickhousePassword` | empty | Password for the connection. |
| `--clickhouseDatabase` | `default` | Database to connect to. |
| `--clickhouseDelay` | `1s` | Interval between sending results to ClickHouse. |
| `--clickhouseDebug` | `false` | Debug the ClickHouse connection. |
| `--clickhouseCompress` | `false` | Compress the ClickHouse connection. |
| `--clickhouseSecure` | `false` | Use TLS for the connection. |
| `--clickhouseSaveFullQuery` | `false` | Save the full packet query and response as JSON. |
| `--clickhouseOutputType` | `0` | Output type, `0`–`4`. See [output types](/outputs/#output-types). |
| `--clickhouseBatchSize` | `100000` | Minimum capacity of the send cache. Set close to your queries per second to avoid reallocation. |
| `--clickhouseWorkers` | `1` | Number of ClickHouse output workers. |
| `--clickhouseWorkerChannelSize` | `100000` | Channel size per worker. |

The general `--skipTLSVerification` option applies to this module as well.

## Retention policy

The default retention for the ClickHouse tables is 30 days. You can change it when building the
containers with `./autobuild.sh`.

ClickHouse has no internal ingest timestamp, so the TTL looks at the packet date from the `pcap`
file. When importing old pcaps, ClickHouse may start removing data as it is written and you will see
nothing in Grafana. To fix that, set the TTL to a day older than the earliest packet in your capture.

To change the TTL manually, connect with `clickhouse-client` and run:

```sql
ALTER TABLE DNS_LOG MODIFY TTL DnsDate + INTERVAL 90 DAY;
```

That only changes the TTL for the raw DNS log data, which is the bulk of your capacity. To adjust
every aggregation table too:

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

{{< callout type="warning" title="Newer ClickHouse versions" >}}
In recent ClickHouse releases the `.inner` tables no longer share a name with their aggregation
views. To modify the TTL you have to find the table names in UUID format with `SHOW TABLES` and
repeat the `ALTER` command against those UUIDs.
{{< /callout >}}

## SAMPLE in SELECT queries

The main tables created by `clickhouse/tables.sql` (`DNS_LOG`) can sample results down, because each
DNS question carries a semi-unique UUID. See the ClickHouse
[SAMPLE documentation](https://clickhouse.tech/docs/en/sql-reference/statements/select/sample/) for
what that enables.

## Useful queries

Unique domains visited over the past 24 hours:

```sql
-- using the domain_count table
SELECT DISTINCT Question FROM DNS_DOMAIN_COUNT WHERE t > Now() - toIntervalHour(24)

-- only the number
SELECT count(DISTINCT Question) FROM DNS_DOMAIN_COUNT WHERE t > Now() - toIntervalHour(24)

-- memory usage of the above query, in bytes
SELECT memory_usage FROM system.query_log WHERE query_kind='Select' AND arrayExists(x-> x='default.DNS_DOMAIN_COUNT', tables) ORDER BY event_time DESC LIMIT 1 format Vertical

-- memory usage of a specific query by ID
SELECT sum(memory_usage) FROM system.query_log WHERE initial_query_id = '8de8fe3c-d46a-4a32-83da-4f4ba4dc49e5' format Vertical
```
