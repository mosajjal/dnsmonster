---
title: "ClickHouse Cloud"
linkTitle: "ClickHouse Cloud"
weight: 6
date: 2022-06-01
description: >
  use dnsmonster with ClickHouse Cloud
---

[ClickHouse Cloud](https://clickhouse.com/cloud/) is a Serverless ClickHouse offering by the ClickHouse team. In this small tutorial I'll go through the steps of building your DNS monitoring with it. At the time of writing this post, ClickHouse Cloud is in preview and some of the features might change over time.

## Create a ClickHouse Cluster

First, let's create a ClickHouse instance by signing up and logging into [ClickHouse Cloud portal](https://clickhouse.cloud) and clicking on "New Service" on the top right corner. You will be asked to provide a name and a region for your database. For the purpose of this tutorial, I will put the name of the database as `dnsmonster` in `us-east-2` region. There's a good chance that other parameters will be present when you define your cluster such as size and number of servers, but overall everything should look pretty much the same 

After clicking on create, you'll see the connection settings for your instance. the default username to login is `default` and the password is generated randomly. Save that password for a later use since the portal won't show it forever! 

And that's it! You have a fully managed ClickHouse cluster running in AWS. Now let's create our tables and views using the credentials we just got.

## Create and configure Tables

when you checkout `dnsmonster` repository from GitHub, there is a [replicated table](https://raw.githubusercontent.com/mosajjal/dnsmonster/main/clickhouse/tables_replicated.sql) file with the table definitions suited for ClickHouse cloud. note that the "traditional" table design won't work on ClickHouse cloud since the managed cluster won't allow non-replicated tables. This policy has been put in place to ensure the high availability and integrity of the tables' data. Download the `.sql` file and save it anywhere on your disk.
for example, `/tmp/tables_replicated.sql`. Now let's use `clickhouse-client` tool to create the tables.

```sh
clickhouse-client --host INSTANCEID.REGION.PROVIDER.clickhouse.cloud --secure --port 9440 --password RANDOM_PASSWORD --multiquery < /tmp/tables_replicated.sql
```

replace the all caps variables with your server instance and this should create your primary tables. Everything should be in place for us to use `dnsmonster`. Now we can point the `dnsmonster` service to the ClickHouse instance and it should work without any issues.

```sh
dnsmonster --devName=lo \                                                                                                                                                         0.351s 19:41
          --packetHandlerCount=8 \
          --clickhouseAddress=INSTANCEID.REGION.PROVIDER.clickhouse.cloud:9440 \
          --clickhouseOutputType=1 \
          --clickhouseBatchSize=7000 \
          --clickhouseWorkers=16 \
          --clickhouseSecure \
          --clickhouseUsername=default \
          --clickhousePassword="RANDOM_PASSWORD" \
          --clickhouseCompress \
          --serverName=my_dnsmonster \
          --maskSize4=16 \
          --maskSize6=64
```

Compressing the ClickHouse `INSERT` connection (`--clickhouseCompress`) will make it efficient and fast. I've gotten better result by turning it on. Keep in mind that the tweaking of the packetHandlerCount as well as number of ClickHouse workers, batch size etc. will have a major impact on the overall performance. In my test, I've been able to exceed ~250,000 packets per seconds easily on my fibre connection. Keep in mind that you can substitute command line arguments with environment variables or a config file. Refer to the Configuration section of the documents for more info. 


## Configuring Grafana and dashboards

Now that the data is being pushed into ClickHouse, you can leverage Grafana with the pre-built dashboard to help you gain visibility over your data. Let's start with running an instance of Grafana in a docker container.

```sh
docker run --name dnsmonster_grafana -p 3000:3000 grafana/grafana:8.4.3
```

then browse to `localhost:3000` with `admin` as both username and password, and install the ClickHouse plugin for Grafana. There are two choices in Grafana store, so both of them should work file out of the box, I've tested [Altinity plugin for ClickHouse](https://grafana.com/grafana/plugins/vertamedia-clickhouse-datasource/) but there's also an official [ClickHouse Grafana Plugin](https://grafana.com/grafana/plugins/grafana-clickhouse-datasource/) to choose from.

After installing the plugin, you can add your ClickHouse server as a datasource using the same address, port and the password you used to run dnsmonster. After connecting Grafana to ClickHouse, you can import the pre-built dashboard from [here](https://raw.githubusercontent.com/mosajjal/dnsmonster/main/grafana/panel.json) either via the GUI or the CLI. Once your dashboard is imported, you can point it to your datasource address and most panels should start showing data. most, but not all.  

One final step to make sure everything is running smoothly, is to `INSERT` the dictionaries. Download the 4 dictonary files located [here](https://github.com/mosajjal/dnsmonster/tree/main/clickhouse/dictionaries) either manually or by cloning the git repo. I'll assume that they're in your `/tmp/` directory. Now let's go back to `clickhouse-client` and quickly make that happen

```sql
clickhouse-client --host INSTANCEID.REGION.PROVIDER.clickhouse.cloud --secure --port 9440 --password RANDOM_PASSWORD 
CREATE DICTIONARY dns_class (Id Uint64, Name String) PRIMARY KEY Id LAYOUT(FLAT()) SOURCE(HTTP(url "https://raw.githubusercontent.com/mosajjal/dnsmonster/main/clickhouse/dictionaries/dns_class.tsv" format TSV)) LIFETIME(MIN 0 MAX 0)
CREATE DICTIONARY dns_opcode (Id Uint64, Name String) PRIMARY KEY Id LAYOUT(FLAT()) SOURCE(HTTP(url "https://raw.githubusercontent.com/mosajjal/dnsmonster/main/clickhouse/dictionaries/dns_opcode.tsv" format TSV))  LIFETIME(MIN 0 MAX 0) 
CREATE DICTIONARY dns_response (Id Uint64, Name String) PRIMARY KEY Id LAYOUT(FLAT()) SOURCE(HTTP(url "https://raw.githubusercontent.com/mosajjal/dnsmonster/main/clickhouse/dictionaries/dns_response.tsv" format TSV))  LIFETIME(MIN 0 MAX 0) 
CREATE DICTIONARY dns_type (Id Uint64, Name String) PRIMARY KEY Id LAYOUT(FLAT()) SOURCE(HTTP(url "https://raw.githubusercontent.com/mosajjal/dnsmonster/main/clickhouse/dictionaries/dns_type.tsv" format TSV)) LIFETIME(MIN 0 MAX 0) 
```

And that's about it. With above commands, the full stack of Grafana, ClickHouse and dnsmonster should work perfectly. No more managing ClickHouse clusters manually! You can also combine this with the Kubernetes tutorial and provide a cloud-native, serverless DNS monitoring platform at scale.
