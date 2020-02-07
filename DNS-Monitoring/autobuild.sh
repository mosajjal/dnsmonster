#!/bin/sh
IFACE=INTERFACE

# requirements
yum install -y libpcap podman

mkdir /opt/dnszeppelin/
mkdir /data/clickhouse-data/
mkdir /data/clickhouse-data/logs/
mkdir /data/clickhouse-data/database-data/
cp dnszeppelin-clickhouse /opt/dnszeppelin/
cp clickhouseretention.sh /etc/cron.daily/
chmod +x /opt/dnszeppelin/dnszeppelin-clickhouse
chmod +x /etc/cron.daily/clickhouseretention.sh

# clickhouse-server config
podman run -d --name dns-clickhouse-server --privileged -p 9000:9000 -p 8123:8123 --ulimit nofile=262144:262144 --volume=/data/clickhouse-data/logs/:/var/log/clickhouse-server/ --volume=/data/clickhouse-data/database-data:/var/lib/clickhouse docker.io/yandex/clickhouse-server
podman cp dns_dictionary.xml dns-clickhouse-server:/etc/clickhouse-server/dns_dictionary.xml
podman cp dictionaries/dns_class.tsv dns-clickhouse-server:/opt/dictionaries/
podman cp dictionaries/dns_opcode.tsv dns-clickhouse-server:/opt/dictionaries/
podman cp dictionaries/dns_responce.tsv dns-clickhouse-server:/opt/dictionaries/
podman cp dictionaries/dns_type.tsv dns-clickhouse-server:/opt/dictionaries/

# clickhouse server service
cp clickhouse.service /etc/systemd/system/clickhouse.service
systemctl enable --now clickhouse.service

#diable SElinux
setenforce 0

# create the tables
podman pull docker.io/yandex/clickhouse-client
cat tables.sql | podman run -i -a stdin,stdout,stderr --rm --net=host yandex/clickhouse-client -h 127.0.0.1 --multiquery

# dnszeppelin-clickhouse example (only for reference)
# ./dnszeppelin-clickhouse -serverName 127.0.0.1 -clickhouseAddress localhost:9000 -devName enp25s0f0 -batchSize 1000000

# write this to /etc/systemd/system/dnszeppelin.service
cp "dnszeppelin@.service" "/etc/systemd/system/dnszeppelin@.service"
cp dnszeppelin-logrotate /etc/logrotate.d/dnszeppelin-logrotate
systemctl enable --now dnszeppelin@$IFACE.service

# test
podman run -i -a stdin,stdout,stderr --rm --net=host  yandex/clickhouse-client -h 127.0.0.1 -q "select count (*) from (select * from DNS_LOG)"
