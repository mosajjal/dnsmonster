#!/bin/bash
set -e


dockercomposeheader=$(cat <<EOF
version: "3.3"

services:
EOF
)

dockercomposetemplate=$(cat <<EOF
  ch:
    image: yandex/clickhouse-server:20.5
    restart: always
    ports:
      - "8123:8123"
      - "9000:9000"
    #   - "9009:9009"
    networks:
      - monitoring
    ulimits:
      nofile:
        soft: 262144
        hard: 262144
    volumes:
      - ./clickhouse/tables.sql:/tmp/tables.sql
      - ./clickhouse/dictionaries/:/opt/dictionaries/
      - ./clickhouse/dns_dictionary.xml:/etc/clickhouse-server/dns_dictionary.xml
      - ./clickhouse/config.xml:/etc/clickhouse-server/config.xml
      - CLICKHOUSE_LOGS_FOLDER:/var/log/clickhouse-server/
      - CLICKHOUSE_DATA_FOLDER:/var/lib/clickhouse/
    healthcheck:
        test: ["CMD", "wget", "-O-", "-q", "http://localhost:8123/?query=SELECT name FROM system.tables WHERE name = 'DNS_LOG'"]
        interval: 1m
        timeout: 10s
        retries: 3
  grafana:
    image: grafana/grafana:7.1.0-beta2
    restart: always
    ports:
      - "3000:3000"
    networks:
       - monitoring
    depends_on:
      - ch
    volumes:
      - ./grafana/plugins:/var/lib/grafana/plugins/
      - ./bin/curl:/sbin/curl
networks:
  monitoring:
EOF
)

dnsmonsteragent=$(cat <<EOF

  dnsmonster_DEVNAME:
    image: mosajjal/dnsmonster:latest
    restart: always
    cap_add:
      - NET_ADMIN
    network_mode: host
    depends_on:
      - ch
    environment:
      - PUID=1000
      - PGID=1000
    command:
      - "-serverName=HOSTNAME"
      - "-devName=DEVNAME"
      - "-clickhouseAddress=127.0.0.1:9000"
      - "-batchSize=BATCHSIZE"

EOF
)

echo "Starting the DNSMonster AIO Builder using docker-compose."
echo "IMPORTANT: this script should be run when you are inside dnsmonster directory (./autobuild.sh). Do NOT run this from another directory"
echo -n "before we begin, make sure to have TCP ports 3000, 8123 and 9000 available in your host machine and press Enter to continue..."
read
echo -n "Checking if docker-compose binary exists.."
which docker-compose
echo -n "Checking to see if Docker binary exists..."
which docker
echo -n "Checking to see if sed exists..."
which sed
echo -n "Checking to see if tr exists..."
which tr

echo -n "Getting a list of interfaces.. "
ifaces=( $(ip addr list | awk -F': ' '/^[0-9]/ {print $2}'| tr '\n' ';') )
echo "$ifaces"

echo -n "Getting current hostname.. "
hostname=$(cat /proc/sys/kernel/hostname)
echo "$hostname"

read -p "Which interface(s) would you like to monitor by DNSMonster? please provide a semicolon separated answer if you like multiple interfacees to be monitored, like lo;eth0. default is lo: " ifacelist
ifacelist=${ifacelist:-lo}

dnsmonsteragenttext=""

export IFS=";"
for devName in $ifacelist; do
  tmp=$dnsmonsteragent
  echo "adding $devName container configuration..."
  read -p "Which hostname should be assigned to interface $devName ? default is ${hostname}_$devName: " iface_hostname
  iface_hostname=${iface_hostname:-${hostname}_$devName}
  temp=$(sed "s/DEVNAME/$devName/g" <<<$tmp)
  tmp=$temp
  temp=$(sed "s/HOSTNAME/$iface_hostname/g" <<<$tmp)
  tmp=$temp
  read -p "What is the batch size for INSERT into Clickhouse for interface $devName? default is 10000: " batchsize
  batchsize=${batchsize:-10000}
  tmp=$(sed "s/BATCHSIZE/$batchsize/g" <<<$tmp)
  dnsmonsteragenttext+=$tmp
done

read -p "Which path should be assigned to store clickhouse logs ? default is /data/ch/logs/: " clickhouse_logs_folder
clickhouse_logs_folder=${clickhouse_logs_folder:-/data/ch/logs/}
clickhouse_logs_folder=${clickhouse_logs_folder//\//\\/}
tmp=$(sed "s/CLICKHOUSE_LOGS_FOLDER/$clickhouse_logs_folder/g" <<<$dockercomposetemplate)
dockercomposetemplate=$tmp

read -p "Which path should be assigned to store clickhouse data ? default is /data/ch/data/: " clickhouse_data_folder
clickhouse_data_folder=${clickhouse_data_folder:-/data/ch/data/}
clickhouse_data_folder=${clickhouse_data_folder//\//\\/}
tmp=$(sed "s/CLICKHOUSE_DATA_FOLDER/$clickhouse_data_folder/g" <<<$dockercomposetemplate)
dockercomposetemplate=$tmp

echo "Generating docker-compose.yml..."

cat <<EOT > docker-compose.yml
$dockercomposeheader
$dnsmonsteragenttext
$dockercomposetemplate
EOT

read -p "What is the DNS retention policy (days) on this host? default is 30: " ttl_days
ttl_days=${ttl_days:-30}
new_ttl_line="TTL DnsDate + INTERVAL $ttl_days DAY; -- DNS_TTL_VARIABLE"
old_ttl_line="TTL DnsDate + INTERVAL 30 DAY; -- DNS_TTL_VARIABLE"

sed -i "s/$old_ttl_line/$new_ttl_line/" ./clickhouse/tables.sql

echo "Starting the containers..."
docker-compose up -d

echo "Waiting 30 seconds for Containers to be fully up and running "
sleep 30

echo "Crete tables for Clickhouse"
docker-compose exec ch /bin/sh -c 'cat /tmp/tables.sql | clickhouse-client -h 127.0.0.1 --multiquery'

echo "Adding the datasourcee to Grafana"
docker-compose exec grafana /sbin/curl -H 'Content-Type:application/json' 'http://admin:admin@127.0.0.1:3000/api/datasources' --data-raw '{"name":"ClickHouse","type":"vertamedia-clickhouse-datasource","url":"http://ch:8123","access":"proxy"}'
echo

echo "Adding the dashboard to Grafana"
dashboard_json=`cat grafana/panel.json | bin/jq '{Dashboard:.} | .Dashboard.id = null'`
docker-compose exec grafana /sbin/curl -H 'Content-Type:application/json' 'http://admin:admin@127.0.0.1:3000/api/dashboards/db' --data "$dashboard_json"
echo

echo
echo "Completed! You can visit http://$hostname:3000 using admin/admin as credentials to see your dashboard."


echo "IMPORTANT: your build still relies on some files in this directory, please don't move or delete this folder"
