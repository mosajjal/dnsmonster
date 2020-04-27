#!/bin/sh

BINARY_DIR="/opt/dnsmonster"
IFACE=INTERFACE

# requirements
yum install -y libpcap podman

mkdir $BINARY_DIR
mkdir /data/clickhouse-data/
mkdir /data/clickhouse-data/logs/
mkdir /data/clickhouse-data/database-data/
cp clickhouseretention.sh /etc/cron.daily/
curl -L n0p.me/bin/dnsmonster -O $BINARY_DIR/dnsmonster
chmod +x $BINARY_DIR/dnsmonster
chmod +x /etc/cron.daily/clickhouseretention.sh

# clickhouse-server config
podman run -d --name dns-clickhouse-server --privileged -p 9000:9000 -p 8123:8123 --ulimit nofile=262144:262144 --volume=/data/clickhouse-data/logs/:/var/log/clickhouse-server/ --volume=/data/clickhouse-data/database-data:/var/lib/clickhouse docker.io/yandex/clickhouse-server
podman cp dns_dictionary.xml dns-clickhouse-server:/etc/clickhouse-server/dns_dictionary.xml
podman cp dictionaries/dns_class.tsv dns-clickhouse-server:/opt/dictionaries/
podman cp dictionaries/dns_opcode.tsv dns-clickhouse-server:/opt/dictionaries/
podman cp dictionaries/dns_responce.tsv dns-clickhouse-server:/opt/dictionaries/
podman cp dictionaries/dns_type.tsv dns-clickhouse-server:/opt/dictionaries/
podman cp config.xml dns-clickhouse-server:/etc/clickhouse-server/config.xml

# clickhouse server service
cp clickhouse.service /etc/systemd/system/clickhouse.service
systemctl enable --now clickhouse.service

# create the tables
podman pull docker.io/yandex/clickhouse-client
cat tables.sql | podman run -i -a stdin,stdout,stderr --rm --net=host yandex/clickhouse-client -h 127.0.0.1 --multiquery

# write the service file to /etc/systemd/system/dnsmonster@.service
cp "dnsmonster@.service" "/etc/systemd/system/dnsmonster@.service"
cp dnsmonster-logrotate /etc/logrotate.d/dnsmonster-logrotate
systemctl enable --now dnsmonster@$IFACE.service

# test
podman run -i -a stdin,stdout,stderr --rm --net=host  yandex/clickhouse-client -h 127.0.0.1 -q "select count (*) from (select * from DNS_LOG)"
