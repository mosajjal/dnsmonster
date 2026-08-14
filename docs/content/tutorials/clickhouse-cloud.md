+++
title = "ClickHouse Cloud"
description = "Run dnsmonster against a serverless ClickHouse Cloud cluster — replicated tables, dictionaries and the Grafana dashboard."
weight = 2
+++

[ClickHouse Cloud](https://clickhouse.com/cloud/) is a serverless ClickHouse offering from the
ClickHouse team. This walkthrough builds DNS monitoring on top of it.

## Create a ClickHouse cluster

Sign up and log in to the [ClickHouse Cloud portal](https://clickhouse.cloud), then click "New
Service" in the top right corner. You will be asked for a name and a region. This tutorial uses
`dnsmonster` in `us-east-2`. Other parameters may appear when defining your cluster — size, number
of servers — but the rest of the flow is the same.

After clicking create, you get the connection settings for the instance. The default username is
`default` and the password is generated randomly. **Save that password** — the portal will not show
it forever.

That is a fully managed ClickHouse cluster running in AWS. Now create the tables and views.

## Create and configure tables

The repository ships a
[replicated table](https://raw.githubusercontent.com/mosajjal/dnsmonster/main/clickhouse/tables_replicated.sql)
definition suited to ClickHouse Cloud. The traditional table design will not work there, because the
managed cluster does not allow non-replicated tables — a policy that protects availability and data
integrity.

Download the `.sql` file, save it somewhere such as `/tmp/tables_replicated.sql`, and create the
tables with `clickhouse-client`:

```sh
clickhouse-client --host INSTANCEID.REGION.PROVIDER.clickhouse.cloud --secure --port 9440 --password RANDOM_PASSWORD --multiquery < /tmp/tables_replicated.sql
```

Replace the all-caps values with your own. That creates the primary tables. Now point `dnsmonster`
at the instance:

```sh
dnsmonster --devName lo \
          --packetHandlerCount 8 \
          --clickhouseAddress INSTANCEID.REGION.PROVIDER.clickhouse.cloud:9440 \
          --clickhouseOutputType 1 \
          --clickhouseBatchSize 7000 \
          --clickhouseWorkers 16 \
          --clickhouseSecure \
          --clickhouseUsername default \
          --clickhousePassword "RANDOM_PASSWORD" \
          --clickhouseCompress \
          --serverName my_dnsmonster \
          --maskSize4 16 \
          --maskSize6 64
```

Compressing the `INSERT` connection (`--clickhouseCompress`) makes it efficient and fast, and gives
better results in testing. Tuning `packetHandlerCount`, the ClickHouse worker count and the batch
size has a major impact on overall performance — this configuration exceeded ~250,000 packets per
second on a fibre connection. You can substitute the command line arguments with environment
variables or a config file; see [configuration](/configuration/).

## Configure Grafana and dashboards

With data flowing into ClickHouse, Grafana and the pre-built dashboard give you visibility. Start
with a Grafana container:

```sh
docker run --name dnsmonster_grafana -p 3000:3000 grafana/grafana:8.4.3
```

Browse to `localhost:3000` with `admin` as both username and password, and install a ClickHouse
plugin. Both options in the Grafana store work — this walkthrough used the
[Altinity plugin](https://grafana.com/grafana/plugins/vertamedia-clickhouse-datasource/), and there
is also an [official ClickHouse plugin](https://grafana.com/grafana/plugins/grafana-clickhouse-datasource/).

After installing, add your ClickHouse server as a datasource using the same address, port and
password you gave `dnsmonster`. Then import the pre-built dashboard from
[panel.json](https://raw.githubusercontent.com/mosajjal/dnsmonster/main/grafana/panel.json), through
the GUI or the CLI. Point it at your datasource and most panels start showing data. Most, but not
all.

The final step is inserting the dictionaries. Download the four dictionary files from
[the repository](https://github.com/mosajjal/dnsmonster/tree/main/clickhouse/dictionaries), then:

```sql
CREATE DICTIONARY dns_class (Id Uint64, Name String) PRIMARY KEY Id LAYOUT(FLAT()) SOURCE(HTTP(url "https://raw.githubusercontent.com/mosajjal/dnsmonster/main/clickhouse/dictionaries/dns_class.tsv" format TSV)) LIFETIME(MIN 0 MAX 0);
CREATE DICTIONARY dns_opcode (Id Uint64, Name String) PRIMARY KEY Id LAYOUT(FLAT()) SOURCE(HTTP(url "https://raw.githubusercontent.com/mosajjal/dnsmonster/main/clickhouse/dictionaries/dns_opcode.tsv" format TSV)) LIFETIME(MIN 0 MAX 0);
CREATE DICTIONARY dns_response (Id Uint64, Name String) PRIMARY KEY Id LAYOUT(FLAT()) SOURCE(HTTP(url "https://raw.githubusercontent.com/mosajjal/dnsmonster/main/clickhouse/dictionaries/dns_response.tsv" format TSV)) LIFETIME(MIN 0 MAX 0);
CREATE DICTIONARY dns_type (Id Uint64, Name String) PRIMARY KEY Id LAYOUT(FLAT()) SOURCE(HTTP(url "https://raw.githubusercontent.com/mosajjal/dnsmonster/main/clickhouse/dictionaries/dns_type.tsv" format TSV)) LIFETIME(MIN 0 MAX 0);
```

Run those from a `clickhouse-client` session against the same host:

```sh
clickhouse-client --host INSTANCEID.REGION.PROVIDER.clickhouse.cloud --secure --port 9440 --password RANDOM_PASSWORD
```

That is the full stack: Grafana, ClickHouse and `dnsmonster`, with no ClickHouse clusters to manage.
Combine it with the [Kubernetes tutorial](/tutorials/kubernetes/) for a cloud-native, serverless DNS
monitoring platform at scale.
