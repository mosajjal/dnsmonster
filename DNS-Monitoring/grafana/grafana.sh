#!/bin/sh

podman run -d -p 3000:3000 --name=grafana docker.io/grafana/grafana
# Internet access required here
podman exec grafana grafana-cli plugins install grafana-piechart-panel
podman exec grafana-cli plugins install vertamedia-clickhouse-datasource
