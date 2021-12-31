#!/bin/bash

# Simple script to delete partitions older than 30 days. Useful if TTL functionality is misbehaving or you've altered TTL manually after cearing partitions. 

DAYS_TO_KEEP=31
CLICKHOUSE_ADDRESS=localhost:8123

# get the earliest date of partitions from system tables
EARLIEST_DATE=`curl -G ${CLICKHOUSE_ADDRESS} --data-urlencode "query=SELECT partition FROM system.parts WHERE table = 'DNS_LOG' ORDER BY partition LIMIT 1"`

# latest acceptable date to delete partitions
LATEST_DATE=`date +%Y%m%d -d "-${DAYS_TO_KEEP} days"`

# calculate the number of days between the two
let DIFF=($(date +%s -d ${LATEST_DATE})-$(date +%s -d ${EARLIEST_DATE}))/86400
DIFF=$(($DIFF+$DAYS_TO_KEEP))

for i in `seq $DIFF -1 $DAYS_TO_KEEP`
do
  DATE=`date +%Y%m%d -d "-$i days"`
  echo "Deleting partition ${DATE}"
  curl -X POST -G -H "Transfer-Encoding: chunked" ${CLICKHOUSE_ADDRESS} --data-urlencode "query=ALTER TABLE DNS_LOG DROP PARTITION ${DATE}"
done
