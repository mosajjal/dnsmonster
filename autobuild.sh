#!/bin/sh
docker-compose up -d
sleep 5

# Tables insert into ClickHouse
docker-compose exec ch /bin/sh -c 'cat /tmp/tables.sql | clickhouse-client -h 127.0.0.1 --multiquery'

# Add datasource to Grafana
docker-compose exec grafana /sbin/curl -H 'Content-Type:application/json' 'http://admin:admin@127.0.0.1:3000/api/datasources' --data-raw '{"name":"ClickHouse","type":"vertamedia-clickhouse-datasource","url":"http://ch:8123","access":"proxy"}'

# Import Grafana Dashboard
dashboard_json=`cat grafana/panel.json | bin/jq '{Dashboard:.} | .Dashboard.id = null'`
docker-compose exec grafana /sbin/curl -H 'Content-Type:application/json' 'http://admin:admin@127.0.0.1:3000/api/dashboards/db' --data "$dashboard_json"
