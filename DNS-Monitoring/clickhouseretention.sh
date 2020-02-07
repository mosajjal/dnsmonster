#!/bin/bash
retention=30
basecommand="podman run -i -a stdin,stdout,stderr --rm --net=host  yandex/clickhouse-client -h 127.0.0.1 -q"

$basecommand "SELECT name FROM system.tables WHERE database = 'default'" | grep -v \.inner |
  while IFS= read -r line
  do
    sh -c "$basecommand \"ALTER TABLE $line DELETE WHERE timestamp < NOW() - INTERVAL $retention DAY\""
  done
