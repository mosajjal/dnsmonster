{
    "annotations": {
      "list": [
        {
          "builtIn": 1,
          "datasource": "-- Grafana --",
          "enable": true,
          "hide": true,
          "iconColor": "rgba(0, 211, 255, 1)",
          "name": "Annotations & Alerts",
          "type": "dashboard"
        }
      ]
    },
    "editable": true,
    "gnetId": null,
    "graphTooltip": 0,
    "id": 1,
    "iteration": 1575869667128,
    "links": [],
    "panels": [
      {
        "aliasColors": {},
        "bars": true,
        "dashLength": 10,
        "dashes": false,
        "datasource": "clickhouse",
        "fill": 1,
        "gridPos": {
          "h": 7,
          "w": 24,
          "x": 0,
          "y": 0
        },
        "id": 4,
        "legend": {
          "avg": false,
          "current": false,
          "max": false,
          "min": false,
          "show": false,
          "total": false,
          "values": false
        },
        "lines": false,
        "linewidth": 1,
        "links": [],
        "nullPointMode": "null",
        "percentage": false,
        "pointradius": 5,
        "points": false,
        "renderer": "flot",
        "seriesOverrides": [],
        "spaceLength": 10,
        "stack": true,
        "steppedLine": false,
        "targets": [
          {
            "compiledQuery": "SELECT timestamp, groupArray((Question, count)) FROM (SELECT (intDiv(toUInt32(t), 1) * 1) * 1000  as timestamp, Question, sum(c) as count FROM DNS_DOMAIN_COUNT WHERE DnsDate >= toDate(1506625902) AND t >= toDateTime(1506625902) AND Server IN('default','valparaiso') group by t, Question ORDER BY count desc limit 5 by timestamp) GROUP BY timestamp\n ORDER BY timestamp",
            "database": "default",
            "dateColDataType": "DnsDate",
            "dateLoading": false,
            "dateTimeColDataType": "t",
            "datetimeLoading": false,
            "formattedQuery": "<font color=\"darkorange\">SELECT</font> <font color=\"darkseagreen\">timestamp</font>, <font color=\"navajowhite\">groupArray</font>((Question, <font color=\"navajowhite\">count</font>)) <font color=\"darkorange\">FROM</font> (<font color=\"darkorange\">SELECT</font> <font color=\"darkcyan\">$timeSeries</font>  <font color=\"darkorange\">as</font> <font color=\"darkseagreen\">timestamp</font>, Question, <font color=\"navajowhite\">sum</font>(c) <font color=\"darkorange\">as</font> <font color=\"navajowhite\">count</font> <font color=\"darkorange\">FROM</font> <font color=\"darkcyan\">$table</font> <font color=\"darkorange\">WHERE</font> <font color=\"darkcyan\">$timeFilter</font> <font color=\"yellow\">AND</font> Server <font color=\"darkorange\">IN</font>(<font color=\"darkcyan\">$ServerName</font>) <font color=\"darkorange\">group by</font> t, Question <font color=\"darkorange\">ORDER BY</font> <font color=\"navajowhite\">count</font> <font color=\"darkorange\">desc</font> <font color=\"darkorange\">limit</font> <font color=\"cornflowerblue\">5</font> by <font color=\"darkseagreen\">timestamp</font>) <font color=\"darkorange\">group by</font> <font color=\"darkseagreen\">timestamp</font><br /> <font color=\"darkorange\">order by</font> <font color=\"darkseagreen\">timestamp</font>",
            "interval": "",
            "intervalFactor": 1,
            "query": "SELECT timestamp, groupArray((Question, count)) FROM (SELECT $timeSeries  as timestamp, Question, sum(c) as count FROM DNS_DOMAIN_COUNT WHERE $timeFilter AND Server IN($ServerName) group by t, Question ORDER BY count desc limit 5 by timestamp) GROUP BY timestamp\n ORDER BY timestamp",
            "rawQuery": "SELECT timestamp, groupArray((Question, count)) FROM (SELECT (intDiv(toUInt32(t), 1) * 1) * 1000  as timestamp, Question, sum(c) as count FROM default.DNS_DOMAIN_COUNT WHERE DnsDate >= toDate(1506436944) AND t >= toDateTime(1506436944) AND Server IN('valparaiso') group by t, Question ORDER BY count desc limit 5 by timestamp) group by timestamp  order by timestamp",
            "refId": "A",
            "resultFormat": "time_series",
            "table": "DNS_DOMAIN_COUNT",
            "tableLoading": false,
            "tags": [],
            "targetLists": [
              [
                {
                  "params": [
                    "*"
                  ],
                  "type": "field"
                },
                {
                  "params": [],
                  "type": "count"
                }
              ]
            ]
          }
        ],
        "thresholds": [],
        "timeFrom": null,
        "timeShift": null,
        "title": "Top Queried domains",
        "tooltip": {
          "shared": false,
          "sort": 0,
          "value_type": "individual"
        },
        "type": "graph",
        "xaxis": {
          "buckets": null,
          "mode": "time",
          "name": null,
          "show": true,
          "values": []
        },
        "yaxes": [
          {
            "format": "opm",
            "label": null,
            "logBase": 1,
            "max": null,
            "min": null,
            "show": true
          },
          {
            "format": "short",
            "label": null,
            "logBase": 1,
            "max": null,
            "min": null,
            "show": true
          }
        ],
        "yaxis": {
          "align": false,
          "alignLevel": null
        }
      },
      {
        "aliasColors": {},
        "bars": false,
        "dashLength": 10,
        "dashes": false,
        "datasource": "clickhouse",
        "fill": 1,
        "gridPos": {
          "h": 7,
          "w": 8,
          "x": 0,
          "y": 7
        },
        "id": 3,
        "legend": {
          "avg": false,
          "current": false,
          "max": false,
          "min": false,
          "show": true,
          "total": false,
          "values": false
        },
        "lines": true,
        "linewidth": 1,
        "links": [],
        "nullPointMode": "null",
        "percentage": false,
        "pointradius": 5,
        "points": false,
        "renderer": "flot",
        "seriesOverrides": [],
        "spaceLength": 10,
        "stack": false,
        "steppedLine": false,
        "targets": [
          {
            "compiledQuery": "SELECT (intDiv(toUInt32(timestamp), 2) * 2) * 1000 as t, uniqMerge(UniqueDnsCount)/2 as Count  FROM DNS_DOMAIN_UNIQUE WHERE DnsDate >= toDate(1506625902) AND timestamp >= toDateTime(1506625902) AND Server IN('default','valparaiso')   GROUP BY t ORDER BY t",
            "database": "default",
            "dateColDataType": "DnsDate",
            "dateLoading": false,
            "dateTimeColDataType": "timestamp",
            "datetimeLoading": false,
            "formattedQuery": "<font color=\"darkorange\">SELECT</font> <font color=\"darkcyan\">$timeSeries</font> <font color=\"darkorange\">as</font> t, uniqMerge(UniqueDnsCount)<font color=\"yellow\">/</font><font color=\"darkcyan\">$interval</font> <font color=\"darkorange\">as</font> Count  <font color=\"darkorange\">FROM</font> <font color=\"darkcyan\">$table</font> <font color=\"darkorange\">WHERE</font> <font color=\"darkcyan\">$timeFilter</font> <font color=\"yellow\">AND</font> Server <font color=\"darkorange\">IN</font>(<font color=\"darkcyan\">$ServerName</font>)   <font color=\"darkorange\">GROUP BY</font> t <font color=\"darkorange\">ORDER BY</font> t",
            "intervalFactor": 1,
            "query": "SELECT $timeSeries as t, uniqMerge(UniqueDnsCount)/$interval as Count  FROM DNS_DOMAIN_UNIQUE WHERE $timeFilter AND Server IN($ServerName)   GROUP BY t ORDER BY t",
            "rawQuery": "SELECT (intDiv(toUInt32(timestamp), 2) * 2) * 1000 as t, uniqMerge(UniqueDnsCount)/2 as Count  FROM default.DNS_DOMAIN_UNIQUE WHERE DnsDate >= toDate(1506436944) AND timestamp >= toDateTime(1506436944) AND Server IN('valparaiso')   GROUP BY t ORDER BY t",
            "refId": "A",
            "resultFormat": "time_series",
            "table": "DNS_DOMAIN_UNIQUE",
            "tableLoading": false,
            "tags": [],
            "targetLists": [
              [
                {
                  "params": [
                    "*"
                  ],
                  "type": "field"
                },
                {
                  "params": [],
                  "type": "count"
                }
              ]
            ]
          }
        ],
        "thresholds": [],
        "timeFrom": null,
        "timeShift": null,
        "title": "Unique Domains Queried Per Second",
        "tooltip": {
          "shared": true,
          "sort": 0,
          "value_type": "individual"
        },
        "type": "graph",
        "xaxis": {
          "buckets": null,
          "mode": "time",
          "name": null,
          "show": true,
          "values": []
        },
        "yaxes": [
          {
            "format": "short",
            "label": null,
            "logBase": 1,
            "max": null,
            "min": null,
            "show": true
          },
          {
            "format": "short",
            "label": null,
            "logBase": 1,
            "max": null,
            "min": null,
            "show": true
          }
        ],
        "yaxis": {
          "align": false,
          "alignLevel": null
        }
      },
      {
        "aliasColors": {},
        "bars": false,
        "dashLength": 10,
        "dashes": false,
        "datasource": "clickhouse",
        "fill": 1,
        "gridPos": {
          "h": 7,
          "w": 8,
          "x": 8,
          "y": 7
        },
        "id": 5,
        "legend": {
          "avg": false,
          "current": false,
          "max": false,
          "min": false,
          "show": true,
          "total": false,
          "values": false
        },
        "lines": true,
        "linewidth": 1,
        "links": [],
        "nullPointMode": "null",
        "percentage": false,
        "pointradius": 5,
        "points": false,
        "renderer": "flot",
        "seriesOverrides": [],
        "spaceLength": 10,
        "stack": false,
        "steppedLine": false,
        "targets": [
          {
            "compiledQuery": "SELECT (intDiv(toUInt32(timestamp), 2) * 2) * 1000 as t, avgMerge(AverageSize) AS Bytes FROM DNS_GENERAL_AGGREGATIONS WHERE DnsDate >= toDate(1506625902) AND timestamp >= toDateTime(1506625902) AND Server IN('default','valparaiso')   Group by t order by t",
            "database": "default",
            "dateColDataType": "DnsDate",
            "dateLoading": false,
            "dateTimeColDataType": "timestamp",
            "datetimeLoading": false,
            "formattedQuery": "<font color=\"darkorange\">SELECT</font> <font color=\"darkcyan\">$timeSeries</font> <font color=\"darkorange\">as</font> t, avgMerge(AverageSize) <font color=\"darkorange\">AS</font> Bytes <font color=\"darkorange\">FROM</font> <font color=\"darkcyan\">$table</font> <font color=\"darkorange\">WHERE</font> <font color=\"darkcyan\">$timeFilter</font> <font color=\"yellow\">AND</font> Server <font color=\"darkorange\">IN</font>(<font color=\"darkcyan\">$ServerName</font>)   <font color=\"darkorange\">Group by</font> t <font color=\"darkorange\">order by</font> t",
            "intervalFactor": 1,
            "query": "SELECT $timeSeries as t, avgMerge(AverageSize) AS Bytes FROM DNS_GENERAL_AGGREGATIONS WHERE $timeFilter AND Server IN($ServerName)   Group by t order by t",
            "rawQuery": "SELECT (intDiv(toUInt32(timestamp), 2) * 2) * 1000 as t, avgMerge(AverageSize) AS Bytes FROM default.DNS_GENERAL_AGGREGATIONS WHERE DnsDate >= toDate(1506436944) AND timestamp >= toDateTime(1506436944) AND Server IN('valparaiso')   Group by t order by t",
            "refId": "A",
            "resultFormat": "time_series",
            "table": "DNS_GENERAL_AGGREGATIONS",
            "tableLoading": false,
            "tags": [],
            "targetLists": [
              [
                {
                  "params": [
                    "*"
                  ],
                  "type": "field"
                },
                {
                  "params": [],
                  "type": "count"
                }
              ]
            ]
          }
        ],
        "thresholds": [],
        "timeFrom": null,
        "timeShift": null,
        "title": "Average Packet Size",
        "tooltip": {
          "shared": true,
          "sort": 0,
          "value_type": "individual"
        },
        "type": "graph",
        "xaxis": {
          "buckets": null,
          "mode": "time",
          "name": null,
          "show": true,
          "values": []
        },
        "yaxes": [
          {
            "format": "bytes",
            "label": null,
            "logBase": 1,
            "max": null,
            "min": null,
            "show": true
          },
          {
            "format": "short",
            "label": null,
            "logBase": 1,
            "max": null,
            "min": null,
            "show": true
          }
        ],
        "yaxis": {
          "align": false,
          "alignLevel": null
        }
      },
      {
        "aliasColors": {},
        "bars": false,
        "dashLength": 10,
        "dashes": false,
        "datasource": "clickhouse",
        "fill": 1,
        "gridPos": {
          "h": 7,
          "w": 8,
          "x": 16,
          "y": 7
        },
        "id": 10,
        "legend": {
          "avg": false,
          "current": false,
          "max": false,
          "min": false,
          "show": true,
          "total": false,
          "values": false
        },
        "lines": true,
        "linewidth": 1,
        "links": [],
        "nullPointMode": "null",
        "percentage": false,
        "pointradius": 5,
        "points": false,
        "renderer": "flot",
        "seriesOverrides": [],
        "spaceLength": 10,
        "stack": false,
        "steppedLine": false,
        "targets": [
          {
            "compiledQuery": "SELECT (intDiv(toUInt32(timestamp), 2) * 2) * 1000 as t, sumMerge(TotalSize)*8/2 as \"Bytes/s\" FROM DNS_GENERAL_AGGREGATIONS WHERE DnsDate >= toDate(1506625902) AND timestamp >= toDateTime(1506625902) AND Server IN('default','valparaiso')   Group by t order by t",
            "database": "default",
            "dateColDataType": "DnsDate",
            "dateLoading": false,
            "dateTimeColDataType": "timestamp",
            "datetimeLoading": false,
            "formattedQuery": "SELECT $timeSeries as t, sumMerge(TotalSize)*8/$interval as \"Bytes/s\" FROM $table   WHERE $timeFilter AND Server IN($ServerName)   Group by t order by t",
            "intervalFactor": 1,
            "query": "SELECT $timeSeries as t, sumMerge(TotalSize)*8/$interval as \"Bytes/s\" FROM DNS_GENERAL_AGGREGATIONS WHERE $timeFilter AND Server IN($ServerName)   Group by t order by t",
            "rawQuery": "SELECT (intDiv(toUInt32(timestamp), 2) * 2) * 1000 as t, sumMerge(TotalSize)*8/2 as \"Bytes/s\" FROM default.DNS_GENERAL_AGGREGATIONS   WHERE DnsDate >= toDate(1506436944) AND timestamp >= toDateTime(1506436944) AND Server IN('valparaiso')   Group by t order by t",
            "refId": "A",
            "resultFormat": "time_series",
            "table": "DNS_GENERAL_AGGREGATIONS",
            "tableLoading": false,
            "tags": [],
            "targetLists": [
              [
                {
                  "params": [
                    "*"
                  ],
                  "type": "field"
                },
                {
                  "params": [],
                  "type": "count"
                }
              ]
            ]
          }
        ],
        "thresholds": [],
        "timeFrom": null,
        "timeShift": null,
        "title": "Total Packet Size",
        "tooltip": {
          "shared": true,
          "sort": 0,
          "value_type": "individual"
        },
        "type": "graph",
        "xaxis": {
          "buckets": null,
          "mode": "time",
          "name": null,
          "show": true,
          "values": []
        },
        "yaxes": [
          {
            "format": "bps",
            "label": null,
            "logBase": 1,
            "max": null,
            "min": null,
            "show": true
          },
          {
            "format": "short",
            "label": null,
            "logBase": 1,
            "max": null,
            "min": null,
            "show": true
          }
        ],
        "yaxis": {
          "align": false,
          "alignLevel": null
        }
      },
      {
        "aliasColors": {},
        "bars": false,
        "dashLength": 10,
        "dashes": false,
        "datasource": "clickhouse",
        "fill": 1,
        "gridPos": {
          "h": 6,
          "w": 12,
          "x": 0,
          "y": 14
        },
        "id": 11,
        "legend": {
          "avg": false,
          "current": false,
          "max": false,
          "min": false,
          "show": true,
          "total": false,
          "values": false
        },
        "lines": true,
        "linewidth": 1,
        "links": [],
        "nullPointMode": "null",
        "percentage": false,
        "pointradius": 5,
        "points": false,
        "renderer": "flot",
        "seriesOverrides": [],
        "spaceLength": 10,
        "stack": false,
        "steppedLine": false,
        "targets": [
          {
            "compiledQuery": "SELECT t, groupArray((IPVersion, count/2)) FROM (SELECT (intDiv(toUInt32(timestamp), 2) * 2) * 1000 as t, IPVersion, sum(c) as count FROM DNS_IP_MASK WHERE DnsDate >= toDate(1506625902) AND timestamp >= toDateTime(1506625902) AND Server IN('default','valparaiso')   group by t, IPVersion ORDER BY t) group by t\n order by t",
            "database": "default",
            "dateColDataType": "DnsDate",
            "dateLoading": false,
            "dateTimeColDataType": "timestamp",
            "datetimeLoading": false,
            "formattedQuery": "<font color=\"darkorange\">SELECT</font> t, <font color=\"navajowhite\">groupArray</font>((IPVersion, <font color=\"navajowhite\">count</font><font color=\"yellow\">/</font><font color=\"darkcyan\">$interval</font>)) <font color=\"darkorange\">FROM</font> (<font color=\"darkorange\">SELECT</font> <font color=\"darkcyan\">$timeSeries</font> <font color=\"darkorange\">as</font> t, IPVersion, <font color=\"navajowhite\">sum</font>(c) <font color=\"darkorange\">as</font> <font color=\"navajowhite\">count</font> <font color=\"darkorange\">FROM</font> <font color=\"darkcyan\">$table</font> <font color=\"darkorange\">WHERE</font> <font color=\"darkcyan\">$timeFilter</font> <font color=\"yellow\">AND</font> Server <font color=\"darkorange\">IN</font>(<font color=\"darkcyan\">$ServerName</font>)   <font color=\"darkorange\">group by</font> t, IPVersion <font color=\"darkorange\">ORDER BY</font> t) <font color=\"darkorange\">group by</font> t<br /> <font color=\"darkorange\">order by</font> t",
            "intervalFactor": 2,
            "query": "SELECT t, groupArray((IPVersion, count/$interval)) FROM (SELECT $timeSeries as t, IPVersion, sum(c) as count FROM DNS_IP_MASK WHERE $timeFilter AND Server IN($ServerName)   group by t, IPVersion ORDER BY t) group by t\n order by t",
            "rawQuery": "SELECT t, groupArray((IPVersion, count/4)) FROM (SELECT (intDiv(toUInt32(timestamp), 4) * 4) * 1000 as t, IPVersion, sum(c) as count FROM default.DNS_IP_MASK WHERE DnsDate >= toDate(1506436944) AND timestamp >= toDateTime(1506436944) AND Server IN('valparaiso')   group by t, IPVersion ORDER BY t) group by t  order by t",
            "refId": "A",
            "resultFormat": "time_series",
            "table": "DNS_IP_MASK",
            "tableLoading": false,
            "tags": [],
            "targetLists": [
              [
                {
                  "params": [
                    "*"
                  ],
                  "type": "field"
                },
                {
                  "params": [],
                  "type": "count"
                }
              ]
            ]
          }
        ],
        "thresholds": [],
        "timeFrom": null,
        "timeShift": null,
        "title": "Packet Count byIP Version",
        "tooltip": {
          "shared": true,
          "sort": 0,
          "value_type": "individual"
        },
        "type": "graph",
        "xaxis": {
          "buckets": null,
          "mode": "time",
          "name": null,
          "show": true,
          "values": []
        },
        "yaxes": [
          {
            "format": "pps",
            "label": null,
            "logBase": 1,
            "max": null,
            "min": null,
            "show": true
          },
          {
            "format": "short",
            "label": null,
            "logBase": 1,
            "max": null,
            "min": null,
            "show": true
          }
        ],
        "yaxis": {
          "align": false,
          "alignLevel": null
        }
      },
      {
        "aliasColors": {},
        "bars": false,
        "dashLength": 10,
        "dashes": false,
        "datasource": "clickhouse",
        "fill": 1,
        "gridPos": {
          "h": 6,
          "w": 12,
          "x": 12,
          "y": 14
        },
        "id": 2,
        "legend": {
          "avg": false,
          "current": false,
          "max": false,
          "min": false,
          "show": true,
          "total": false,
          "values": false
        },
        "lines": true,
        "linewidth": 1,
        "links": [],
        "nullPointMode": "null",
        "percentage": false,
        "pointradius": 5,
        "points": false,
        "renderer": "flot",
        "seriesOverrides": [],
        "spaceLength": 10,
        "stack": false,
        "steppedLine": false,
        "targets": [
          {
            "compiledQuery": "SELECT t, groupArray((Protocol, count/2)) FROM (SELECT (intDiv(toUInt32(timestamp), 2) * 2) * 1000 as t, Protocol, sum(c) as count FROM DNS_PROTOCOL WHERE DnsDate >= toDate(1506625902) AND timestamp >= toDateTime(1506625902) AND Server IN('default','valparaiso')   group by t, Protocol ORDER BY t) group by t\n order by t",
            "database": "default",
            "dateColDataType": "DnsDate",
            "dateLoading": false,
            "dateTimeColDataType": "timestamp",
            "datetimeLoading": false,
            "formattedQuery": "<font color=\"darkorange\">SELECT</font> t, <font color=\"navajowhite\">groupArray</font>((Protocol, <font color=\"navajowhite\">count</font><font color=\"yellow\">/</font><font color=\"darkcyan\">$interval</font>)) <font color=\"darkorange\">FROM</font> (<font color=\"darkorange\">SELECT</font> <font color=\"darkcyan\">$timeSeries</font> <font color=\"darkorange\">as</font> t, Protocol, <font color=\"navajowhite\">sum</font>(c) <font color=\"darkorange\">as</font> <font color=\"navajowhite\">count</font> <font color=\"darkorange\">FROM</font> <font color=\"darkcyan\">$table</font> <font color=\"darkorange\">WHERE</font> <font color=\"darkcyan\">$timeFilter</font> <font color=\"yellow\">AND</font> Server <font color=\"darkorange\">IN</font>(<font color=\"darkcyan\">$ServerName</font>)   <font color=\"darkorange\">group by</font> t, Protocol <font color=\"darkorange\">ORDER BY</font> t) <font color=\"darkorange\">group by</font> t<br /> <font color=\"darkorange\">order by</font> t",
            "intervalFactor": 2,
            "query": "SELECT t, groupArray((Protocol, count/$interval)) FROM (SELECT $timeSeries as t, Protocol, sum(c) as count FROM DNS_PROTOCOL WHERE $timeFilter AND Server IN($ServerName)   group by t, Protocol ORDER BY t) group by t\n order by t",
            "rawQuery": "SELECT t, groupArray((Protocol, count/4)) FROM (SELECT (intDiv(toUInt32(timestamp), 4) * 4) * 1000 as t, Protocol, sum(c) as count FROM default.DNS_PROTOCOL WHERE DnsDate >= toDate(1506436944) AND timestamp >= toDateTime(1506436944) AND Server IN('valparaiso')   group by t, Protocol ORDER BY t) group by t  order by t",
            "refId": "A",
            "resultFormat": "time_series",
            "table": "DNS_PROTOCOL",
            "tableLoading": false,
            "tags": [],
            "targetLists": [
              [
                {
                  "params": [
                    "*"
                  ],
                  "type": "field"
                },
                {
                  "params": [],
                  "type": "count"
                }
              ]
            ]
          }
        ],
        "thresholds": [],
        "timeFrom": null,
        "timeShift": null,
        "title": "Packet Count by Transport Protocol",
        "tooltip": {
          "shared": true,
          "sort": 0,
          "value_type": "individual"
        },
        "type": "graph",
        "xaxis": {
          "buckets": null,
          "mode": "time",
          "name": null,
          "show": true,
          "values": []
        },
        "yaxes": [
          {
            "format": "pps",
            "label": null,
            "logBase": 1,
            "max": null,
            "min": null,
            "show": true
          },
          {
            "format": "short",
            "label": null,
            "logBase": 1,
            "max": null,
            "min": null,
            "show": true
          }
        ],
        "yaxis": {
          "align": false,
          "alignLevel": null
        }
      },
      {
        "aliasColors": {},
        "cacheTimeout": null,
        "combine": {
          "label": "Others",
          "threshold": 0
        },
        "datasource": "clickhouse",
        "fontSize": "80%",
        "format": "short",
        "gridPos": {
          "h": 7,
          "w": 12,
          "x": 0,
          "y": 20
        },
        "id": 13,
        "interval": null,
        "legend": {
          "show": false,
          "values": true
        },
        "legendType": "Under graph",
        "links": [],
        "maxDataPoints": 3,
        "nullPointMode": "connected",
        "pieType": "pie",
        "strokeWidth": 1,
        "targets": [
          {
            "compiledQuery": "SELECT 0, groupArray((IP, total)) FROM (SELECT IPv4NumToString(IPPrefix) AS IP,\nsum(c) as total FROM DNS_IP_MASK PREWHERE IPVersion=4 WHERE DnsDate >= toDate(1506625362) AND timestamp >= toDateTime(1506625362) AND Server IN('default','valparaiso')     GROUP BY IPPrefix order by IPPrefix)",
            "database": "default",
            "dateColDataType": "DnsDate",
            "dateLoading": false,
            "dateTimeColDataType": "timestamp",
            "datetimeLoading": false,
            "formattedQuery": "<font color=\"darkorange\">SELECT</font> <font color=\"cornflowerblue\">0</font>, <font color=\"navajowhite\">groupArray</font>((IP, total)) <font color=\"darkorange\">FROM</font> (<font color=\"darkorange\">SELECT</font> <font color=\"navajowhite\">IPv4NumToString</font>(IPPrefix) <font color=\"darkorange\">AS</font> IP,<br /><font color=\"navajowhite\">sum</font>(c) <font color=\"darkorange\">as</font> total <font color=\"darkorange\">FROM</font> <font color=\"darkcyan\">$table</font> <font color=\"darkorange\">PREWHERE</font> IPVersion<font color=\"yellow\">=</font><font color=\"cornflowerblue\">4</font> <font color=\"darkorange\">WHERE</font> <font color=\"darkcyan\">$timeFilter</font> <font color=\"yellow\">AND</font> Server <font color=\"darkorange\">IN</font>(<font color=\"darkcyan\">$ServerName</font>)     <font color=\"darkorange\">GROUP BY</font> IPPrefix <font color=\"darkorange\">order by</font> IPPrefix)",
            "intervalFactor": 1,
            "query": "SELECT 0, groupArray((IP, total)) FROM (SELECT IPv4NumToString(IPPrefix) AS IP,\nsum(c) as total FROM DNS_IP_MASK PREWHERE IPVersion=4 WHERE $timeFilter AND Server IN($ServerName)     GROUP BY IPPrefix order by IPPrefix)",
            "rawQuery": "SELECT 0, groupArray((IP, total)) FROM (SELECT IPv4NumToString(IPPrefix) AS IP, sum(c) as total FROM default.DNS_IP_MASK PREWHERE IPVersion=4 WHERE DnsDate >= toDate(1506436944) AND timestamp >= toDateTime(1506436944) AND Server IN('valparaiso')     GROUP BY IPPrefix order by IPPrefix)",
            "refId": "A",
            "resultFormat": "time_series",
            "table": "DNS_IP_MASK",
            "tableLoading": false,
            "tags": [],
            "targetLists": [
              [
                {
                  "params": [
                    "*"
                  ],
                  "type": "field"
                },
                {
                  "params": [],
                  "type": "count"
                }
              ]
            ]
          }
        ],
        "title": "IPv4 Packet Destination Prefix",
        "type": "grafana-piechart-panel",
        "valueName": "current"
      },
      {
        "aliasColors": {},
        "cacheTimeout": null,
        "combine": {
          "label": "Others",
          "threshold": 0
        },
        "datasource": "clickhouse",
        "fontSize": "80%",
        "format": "short",
        "gridPos": {
          "h": 7,
          "w": 12,
          "x": 12,
          "y": 20
        },
        "id": 12,
        "interval": null,
        "legend": {
          "show": false,
          "values": true
        },
        "legendType": "Under graph",
        "links": [],
        "maxDataPoints": 3,
        "nullPointMode": "connected",
        "pieType": "pie",
        "strokeWidth": 1,
        "targets": [
          {
            "compiledQuery": "SELECT 0, groupArray((IP, total)) FROM (SELECT IPv6NumToString(toFixedString(unhex(hex(IPPrefix)), 16)) AS IP,\nsum(c) as total FROM DNS_IP_MASK PREWHERE IPVersion=6 WHERE DnsDate >= toDate(1506625362) AND timestamp >= toDateTime(1506625362) AND Server IN('default','valparaiso')    GROUP BY IPPrefix order by IPPrefix desc limit 20)",
            "database": "default",
            "dateColDataType": "DnsDate",
            "dateLoading": false,
            "dateTimeColDataType": "timestamp",
            "datetimeLoading": false,
            "formattedQuery": "<font color=\"darkorange\">SELECT</font> <font color=\"cornflowerblue\">0</font>, <font color=\"navajowhite\">groupArray</font>((IP, total)) <font color=\"darkorange\">FROM</font> (<font color=\"darkorange\">SELECT</font> <font color=\"navajowhite\">IPv6NumToString</font>(<font color=\"navajowhite\">toFixedString</font>(<font color=\"navajowhite\">unhex</font>(<font color=\"navajowhite\">hex</font>(IPPrefix)), <font color=\"cornflowerblue\">16</font>)) <font color=\"darkorange\">AS</font> IP,<br /><font color=\"navajowhite\">sum</font>(c) <font color=\"darkorange\">as</font> total <font color=\"darkorange\">FROM</font> <font color=\"darkcyan\">$table</font> <font color=\"darkorange\">PREWHERE</font> IPVersion<font color=\"yellow\">=</font><font color=\"cornflowerblue\">6</font> <font color=\"darkorange\">WHERE</font> <font color=\"darkcyan\">$timeFilter</font> <font color=\"yellow\">AND</font> Server <font color=\"darkorange\">IN</font>(<font color=\"darkcyan\">$ServerName</font>)    <font color=\"darkorange\">GROUP BY</font> IPPrefix <font color=\"darkorange\">order by</font> IPPrefix <font color=\"darkorange\">desc</font> <font color=\"darkorange\">limit</font> <font color=\"cornflowerblue\">20</font>)",
            "intervalFactor": 1,
            "query": "SELECT 0, groupArray((IP, total)) FROM (SELECT IPv6NumToString(toFixedString(unhex(hex(IPPrefix)), 16)) AS IP,\nsum(c) as total FROM DNS_IP_MASK PREWHERE IPVersion=6 WHERE $timeFilter AND Server IN($ServerName)    GROUP BY IPPrefix order by IPPrefix desc limit 20)",
            "rawQuery": "SELECT 0, groupArray((IP, total)) FROM (SELECT IPv6NumToString(toFixedString(unhex(hex(IPPrefix)), 16)) AS IP, sum(c) as total FROM default.DNS_IP_MASK PREWHERE IPVersion=6 WHERE DnsDate >= toDate(1506436944) AND timestamp >= toDateTime(1506436944) AND Server IN('valparaiso')    GROUP BY IPPrefix order by IPPrefix desc limit 20)",
            "refId": "A",
            "resultFormat": "time_series",
            "table": "DNS_IP_MASK",
            "tableLoading": false,
            "tags": [],
            "targetLists": [
              [
                {
                  "params": [
                    "*"
                  ],
                  "type": "field"
                },
                {
                  "params": [],
                  "type": "count"
                }
              ]
            ]
          }
        ],
        "title": "IPv6 Packet Destination Top 20 Prefix",
        "type": "grafana-piechart-panel",
        "valueName": "current"
      },
      {
        "aliasColors": {},
        "bars": false,
        "dashLength": 10,
        "dashes": false,
        "datasource": "clickhouse",
        "fill": 1,
        "gridPos": {
          "h": 7,
          "w": 12,
          "x": 0,
          "y": 27
        },
        "id": 7,
        "legend": {
          "avg": false,
          "current": false,
          "max": false,
          "min": false,
          "show": true,
          "total": false,
          "values": false
        },
        "lines": true,
        "linewidth": 1,
        "links": [],
        "nullPointMode": "null",
        "percentage": false,
        "pointradius": 5,
        "points": false,
        "renderer": "flot",
        "seriesOverrides": [],
        "spaceLength": 10,
        "stack": false,
        "steppedLine": false,
        "targets": [
          {
            "compiledQuery": "SELECT t, groupArray((dictGetString('dns_type', 'Name', toUInt64(Type)), count/2)) FROM (SELECT (intDiv(toUInt32(timestamp), 2) * 2) * 1000 as t, Type, sum(c) as count FROM DNS_TYPE WHERE DnsDate >= toDate(1506625902) AND timestamp >= toDateTime(1506625902) AND Server IN('default','valparaiso')   group by t, Type ORDER BY t) group by t\n order by t",
            "database": "default",
            "dateColDataType": "DnsDate",
            "dateLoading": false,
            "dateTimeColDataType": "timestamp",
            "datetimeLoading": false,
            "formattedQuery": "<font color=\"darkorange\">SELECT</font> t, <font color=\"navajowhite\">groupArray</font>((<font color=\"navajowhite\">dictGetString</font>(<font color=\"lightgreen\">'dns_type'</font>, <font color=\"lightgreen\">'Name'</font>, <font color=\"navajowhite\">toUInt64</font>(<font color=\"darkorange\">Type</font>)), <font color=\"navajowhite\">count</font><font color=\"yellow\">/</font><font color=\"darkcyan\">$interval</font>)) <font color=\"darkorange\">FROM</font> (<font color=\"darkorange\">SELECT</font> <font color=\"darkcyan\">$timeSeries</font> <font color=\"darkorange\">as</font> t, <font color=\"darkorange\">Type</font>, <font color=\"navajowhite\">sum</font>(c) <font color=\"darkorange\">as</font> <font color=\"navajowhite\">count</font> <font color=\"darkorange\">FROM</font> <font color=\"darkcyan\">$table</font> <font color=\"darkorange\">WHERE</font> <font color=\"darkcyan\">$timeFilter</font> <font color=\"yellow\">AND</font> Server <font color=\"darkorange\">IN</font>(<font color=\"darkcyan\">$ServerName</font>)   <font color=\"darkorange\">group by</font> t, <font color=\"darkorange\">Type</font> <font color=\"darkorange\">ORDER BY</font> t) <font color=\"darkorange\">group by</font> t<br /> <font color=\"darkorange\">order by</font> t",
            "intervalFactor": 1,
            "query": "SELECT t, groupArray((dictGetString('dns_type', 'Name', toUInt64(Type)), count/$interval)) FROM (SELECT $timeSeries as t, Type, sum(c) as count FROM DNS_TYPE WHERE $timeFilter AND Server IN($ServerName)   group by t, Type ORDER BY t) group by t\n order by t",
            "rawQuery": "SELECT t, groupArray((dictGetString('dns_type', 'Name', toUInt64(Type)), count/2)) FROM (SELECT (intDiv(toUInt32(timestamp), 2) * 2) * 1000 as t, Type, sum(c) as count FROM default.DNS_TYPE WHERE DnsDate >= toDate(1506436944) AND timestamp >= toDateTime(1506436944) AND Server IN('valparaiso')   group by t, Type ORDER BY t) group by t  order by t",
            "refId": "A",
            "resultFormat": "time_series",
            "table": "DNS_TYPE",
            "tableLoading": false,
            "tags": [],
            "targetLists": [
              [
                {
                  "params": [
                    "*"
                  ],
                  "type": "field"
                },
                {
                  "params": [],
                  "type": "count"
                }
              ]
            ]
          }
        ],
        "thresholds": [],
        "timeFrom": null,
        "timeShift": null,
        "title": "DNS Question Type",
        "tooltip": {
          "shared": true,
          "sort": 0,
          "value_type": "individual"
        },
        "type": "graph",
        "xaxis": {
          "buckets": null,
          "mode": "time",
          "name": null,
          "show": true,
          "values": []
        },
        "yaxes": [
          {
            "format": "pps",
            "label": null,
            "logBase": 1,
            "max": null,
            "min": null,
            "show": true
          },
          {
            "format": "short",
            "label": null,
            "logBase": 1,
            "max": null,
            "min": null,
            "show": true
          }
        ],
        "yaxis": {
          "align": false,
          "alignLevel": null
        }
      },
      {
        "aliasColors": {},
        "bars": false,
        "dashLength": 10,
        "dashes": false,
        "datasource": "clickhouse",
        "fill": 1,
        "gridPos": {
          "h": 7,
          "w": 12,
          "x": 12,
          "y": 27
        },
        "id": 8,
        "legend": {
          "avg": false,
          "current": false,
          "max": false,
          "min": false,
          "show": true,
          "total": false,
          "values": false
        },
        "lines": true,
        "linewidth": 1,
        "links": [],
        "nullPointMode": "null",
        "percentage": false,
        "pointradius": 5,
        "points": false,
        "renderer": "flot",
        "seriesOverrides": [],
        "spaceLength": 10,
        "stack": false,
        "steppedLine": false,
        "targets": [
          {
            "compiledQuery": "SELECT t, groupArray((dictGetString('dns_class', 'Name', toUInt64(Class)), count/2)) FROM (SELECT (intDiv(toUInt32(timestamp), 2) * 2) * 1000 as t, Class, sum(c) as count FROM DNS_CLASS WHERE DnsDate >= toDate(1506625902) AND timestamp >= toDateTime(1506625902) AND Server IN('default','valparaiso')   group by t, Class ORDER BY t) group by t\n order by t",
            "database": "default",
            "dateColDataType": "DnsDate",
            "dateLoading": false,
            "dateTimeColDataType": "timestamp",
            "datetimeLoading": false,
            "formattedQuery": "<font color=\"darkorange\">SELECT</font> t, <font color=\"navajowhite\">groupArray</font>((<font color=\"navajowhite\">dictGetString</font>(<font color=\"lightgreen\">'dns_class'</font>, <font color=\"lightgreen\">'Name'</font>, <font color=\"navajowhite\">toUInt64</font>(Class)), <font color=\"navajowhite\">count</font><font color=\"yellow\">/</font><font color=\"darkcyan\">$interval</font>)) <font color=\"darkorange\">FROM</font> (<font color=\"darkorange\">SELECT</font> <font color=\"darkcyan\">$timeSeries</font> <font color=\"darkorange\">as</font> t, Class, <font color=\"navajowhite\">sum</font>(c) <font color=\"darkorange\">as</font> <font color=\"navajowhite\">count</font> <font color=\"darkorange\">FROM</font> <font color=\"darkcyan\">$table</font> <font color=\"darkorange\">WHERE</font> <font color=\"darkcyan\">$timeFilter</font> <font color=\"yellow\">AND</font> Server <font color=\"darkorange\">IN</font>(<font color=\"darkcyan\">$ServerName</font>)   <font color=\"darkorange\">group by</font> t, Class <font color=\"darkorange\">ORDER BY</font> t) <font color=\"darkorange\">group by</font> t<br /> <font color=\"darkorange\">order by</font> t",
            "intervalFactor": 1,
            "query": "SELECT t, groupArray((dictGetString('dns_class', 'Name', toUInt64(Class)), count/$interval)) FROM (SELECT $timeSeries as t, Class, sum(c) as count FROM DNS_CLASS WHERE $timeFilter AND Server IN($ServerName)   group by t, Class ORDER BY t) group by t\n order by t",
            "rawQuery": "SELECT t, groupArray((dictGetString('dns_class', 'Name', toUInt64(Class)), count/2)) FROM (SELECT (intDiv(toUInt32(timestamp), 2) * 2) * 1000 as t, Class, sum(c) as count FROM default.DNS_CLASS WHERE DnsDate >= toDate(1506436944) AND timestamp >= toDateTime(1506436944) AND Server IN('valparaiso')   group by t, Class ORDER BY t) group by t  order by t",
            "refId": "A",
            "resultFormat": "time_series",
            "table": "DNS_CLASS",
            "tableLoading": false,
            "tags": [],
            "targetLists": [
              [
                {
                  "params": [
                    "*"
                  ],
                  "type": "field"
                },
                {
                  "params": [],
                  "type": "count"
                }
              ]
            ]
          }
        ],
        "thresholds": [],
        "timeFrom": null,
        "timeShift": null,
        "title": "Query Class",
        "tooltip": {
          "shared": true,
          "sort": 0,
          "value_type": "individual"
        },
        "type": "graph",
        "xaxis": {
          "buckets": null,
          "mode": "time",
          "name": null,
          "show": true,
          "values": []
        },
        "yaxes": [
          {
            "format": "ops",
            "label": null,
            "logBase": 1,
            "max": null,
            "min": null,
            "show": true
          },
          {
            "format": "short",
            "label": null,
            "logBase": 1,
            "max": null,
            "min": null,
            "show": true
          }
        ],
        "yaxis": {
          "align": false,
          "alignLevel": null
        }
      },
      {
        "aliasColors": {},
        "bars": false,
        "dashLength": 10,
        "dashes": false,
        "datasource": "clickhouse",
        "fill": 1,
        "gridPos": {
          "h": 7,
          "w": 12,
          "x": 0,
          "y": 34
        },
        "id": 9,
        "legend": {
          "avg": false,
          "current": false,
          "max": false,
          "min": false,
          "show": true,
          "total": false,
          "values": false
        },
        "lines": true,
        "linewidth": 1,
        "links": [],
        "nullPointMode": "null",
        "percentage": false,
        "pointradius": 5,
        "points": false,
        "renderer": "flot",
        "seriesOverrides": [],
        "spaceLength": 10,
        "stack": false,
        "steppedLine": false,
        "targets": [
          {
            "compiledQuery": "SELECT t, groupArray((dictGetString('dns_responce', 'Name', toUInt64(ResponceCode)), count/2)) FROM (SELECT (intDiv(toUInt32(timestamp), 2) * 2) * 1000 as t, ResponceCode, sum(c) as count FROM DNS_RESPONCECODE WHERE DnsDate >= toDate(1506625902) AND timestamp >= toDateTime(1506625902) AND Server IN('default','valparaiso')   group by t, ResponceCode ORDER BY t) group by t\n order by t",
            "database": "default",
            "dateColDataType": "DnsDate",
            "dateLoading": false,
            "dateTimeColDataType": "timestamp",
            "datetimeLoading": false,
            "formattedQuery": "<font color=\"darkorange\">SELECT</font> t, <font color=\"navajowhite\">groupArray</font>((<font color=\"navajowhite\">dictGetString</font>(<font color=\"lightgreen\">'dns_responce'</font>, <font color=\"lightgreen\">'Name'</font>, <font color=\"navajowhite\">toUInt64</font>(ResponceCode)), <font color=\"navajowhite\">count</font><font color=\"yellow\">/</font><font color=\"darkcyan\">$interval</font>)) <font color=\"darkorange\">FROM</font> (<font color=\"darkorange\">SELECT</font> <font color=\"darkcyan\">$timeSeries</font> <font color=\"darkorange\">as</font> t, ResponceCode, <font color=\"navajowhite\">sum</font>(c) <font color=\"darkorange\">as</font> <font color=\"navajowhite\">count</font> <font color=\"darkorange\">FROM</font> <font color=\"darkcyan\">$table</font> <font color=\"darkorange\">WHERE</font> <font color=\"darkcyan\">$timeFilter</font> <font color=\"yellow\">AND</font> Server <font color=\"darkorange\">IN</font>(<font color=\"darkcyan\">$ServerName</font>)   <font color=\"darkorange\">group by</font> t, ResponceCode <font color=\"darkorange\">ORDER BY</font> t) <font color=\"darkorange\">group by</font> t<br /> <font color=\"darkorange\">order by</font> t",
            "intervalFactor": 1,
            "query": "SELECT t, groupArray((dictGetString('dns_responce', 'Name', toUInt64(ResponceCode)), count/$interval)) FROM (SELECT $timeSeries as t, ResponceCode, sum(c) as count FROM DNS_RESPONCECODE WHERE $timeFilter AND Server IN($ServerName)   group by t, ResponceCode ORDER BY t) group by t\n order by t",
            "rawQuery": "SELECT t, groupArray((dictGetString('dns_responce', 'Name', toUInt64(ResponceCode)), count/2)) FROM (SELECT (intDiv(toUInt32(timestamp), 2) * 2) * 1000 as t, ResponceCode, sum(c) as count FROM default.DNS_RESPONCECODE WHERE DnsDate >= toDate(1506436944) AND timestamp >= toDateTime(1506436944) AND Server IN('valparaiso')   group by t, ResponceCode ORDER BY t) group by t  order by t",
            "refId": "A",
            "resultFormat": "time_series",
            "table": "DNS_RESPONCECODE",
            "tableLoading": false,
            "tags": [],
            "targetLists": [
              [
                {
                  "params": [
                    "*"
                  ],
                  "type": "field"
                },
                {
                  "params": [],
                  "type": "count"
                }
              ]
            ]
          }
        ],
        "thresholds": [],
        "timeFrom": null,
        "timeShift": null,
        "title": "Response code",
        "tooltip": {
          "shared": true,
          "sort": 0,
          "value_type": "individual"
        },
        "type": "graph",
        "xaxis": {
          "buckets": null,
          "mode": "time",
          "name": null,
          "show": true,
          "values": []
        },
        "yaxes": [
          {
            "format": "ops",
            "label": null,
            "logBase": 1,
            "max": null,
            "min": null,
            "show": true
          },
          {
            "format": "short",
            "label": null,
            "logBase": 1,
            "max": null,
            "min": null,
            "show": true
          }
        ],
        "yaxis": {
          "align": false,
          "alignLevel": null
        }
      },
      {
        "aliasColors": {},
        "bars": false,
        "dashLength": 10,
        "dashes": false,
        "datasource": "clickhouse",
        "fill": 1,
        "gridPos": {
          "h": 7,
          "w": 12,
          "x": 12,
          "y": 34
        },
        "id": 6,
        "legend": {
          "avg": false,
          "current": false,
          "max": false,
          "min": false,
          "show": true,
          "total": false,
          "values": false
        },
        "lines": true,
        "linewidth": 1,
        "links": [],
        "nullPointMode": "null",
        "percentage": false,
        "pointradius": 5,
        "points": false,
        "renderer": "flot",
        "seriesOverrides": [
          {
            "alias": "0"
          }
        ],
        "spaceLength": 10,
        "stack": false,
        "steppedLine": false,
        "targets": [
          {
            "compiledQuery": "SELECT t, groupArray((dictGetString('dns_opcode', 'Name', toUInt64(OpCode)), count/2)) FROM (SELECT (intDiv(toUInt32(timestamp), 2) * 2) * 1000 as t, OpCode, sum(c) as count FROM DNS_OPCODE WHERE DnsDate >= toDate(1506625902) AND timestamp >= toDateTime(1506625902) AND Server IN('default','valparaiso')   group by t, OpCode ORDER BY t) group by t\n order by t",
            "database": "default",
            "dateColDataType": "DnsDate",
            "dateLoading": false,
            "dateTimeColDataType": "timestamp",
            "datetimeLoading": false,
            "formattedQuery": "<font color=\"darkorange\">SELECT</font> t, <font color=\"navajowhite\">groupArray</font>((<font color=\"navajowhite\">dictGetString</font>(<font color=\"lightgreen\">'dns_opcode'</font>, <font color=\"lightgreen\">'Name'</font>, <font color=\"navajowhite\">toUInt64</font>(OpCode)), <font color=\"navajowhite\">count</font><font color=\"yellow\">/</font><font color=\"darkcyan\">$interval</font>)) <font color=\"darkorange\">FROM</font> (<font color=\"darkorange\">SELECT</font> <font color=\"darkcyan\">$timeSeries</font> <font color=\"darkorange\">as</font> t, OpCode, <font color=\"navajowhite\">sum</font>(c) <font color=\"darkorange\">as</font> <font color=\"navajowhite\">count</font> <font color=\"darkorange\">FROM</font> <font color=\"darkcyan\">$table</font> <font color=\"darkorange\">WHERE</font> <font color=\"darkcyan\">$timeFilter</font> <font color=\"yellow\">AND</font> Server <font color=\"darkorange\">IN</font>(<font color=\"darkcyan\">$ServerName</font>)   <font color=\"darkorange\">group by</font> t, OpCode <font color=\"darkorange\">ORDER BY</font> t) <font color=\"darkorange\">group by</font> t<br /> <font color=\"darkorange\">order by</font> t",
            "interval": "",
            "intervalFactor": 2,
            "query": "SELECT t, groupArray((dictGetString('dns_opcode', 'Name', toUInt64(OpCode)), count/$interval)) FROM (SELECT $timeSeries as t, OpCode, sum(c) as count FROM DNS_OPCODE WHERE $timeFilter AND Server IN($ServerName)   group by t, OpCode ORDER BY t) group by t\n order by t",
            "rawQuery": "SELECT t, groupArray((dictGetString('dns_opcode', 'Name', toUInt64(OpCode)), count/4)) FROM (SELECT (intDiv(toUInt32(timestamp), 4) * 4) * 1000 as t, OpCode, sum(c) as count FROM default.DNS_OPCODE WHERE DnsDate >= toDate(1506436944) AND timestamp >= toDateTime(1506436944) AND Server IN('valparaiso')   group by t, OpCode ORDER BY t) group by t  order by t",
            "refId": "A",
            "resultFormat": "time_series",
            "table": "DNS_OPCODE",
            "tableLoading": false,
            "tags": [],
            "targetLists": [
              [
                {
                  "params": [
                    "*"
                  ],
                  "type": "field"
                },
                {
                  "params": [],
                  "type": "count"
                }
              ]
            ]
          }
        ],
        "thresholds": [],
        "timeFrom": null,
        "timeShift": null,
        "title": "OpCode Received",
        "tooltip": {
          "shared": true,
          "sort": 0,
          "value_type": "individual"
        },
        "type": "graph",
        "xaxis": {
          "buckets": null,
          "mode": "time",
          "name": null,
          "show": true,
          "values": []
        },
        "yaxes": [
          {
            "format": "ops",
            "label": null,
            "logBase": 1,
            "max": null,
            "min": null,
            "show": true
          },
          {
            "format": "short",
            "label": null,
            "logBase": 1,
            "max": null,
            "min": null,
            "show": true
          }
        ],
        "yaxis": {
          "align": false,
          "alignLevel": null
        }
      },
      {
        "aliasColors": {},
        "bars": false,
        "dashLength": 10,
        "dashes": false,
        "datasource": "clickhouse",
        "fill": 1,
        "gridPos": {
          "h": 7,
          "w": 12,
          "x": 0,
          "y": 41
        },
        "id": 14,
        "legend": {
          "avg": false,
          "current": false,
          "max": false,
          "min": false,
          "show": true,
          "total": false,
          "values": false
        },
        "lines": true,
        "linewidth": 1,
        "links": [],
        "nullPointMode": "null",
        "percentage": false,
        "pointradius": 5,
        "points": false,
        "renderer": "flot",
        "seriesOverrides": [],
        "spaceLength": 10,
        "stack": false,
        "steppedLine": false,
        "targets": [
          {
            "compiledQuery": "SELECT (intDiv(toUInt32(timestamp), 2) * 2) * 1000 as t, sumMerge(EdnsCount)/2 as Edns0Present FROM DNS_EDNS WHERE DnsDate >= toDate(1506625902) AND timestamp >= toDateTime(1506625902) AND Server IN('default','valparaiso')   GROUP BY t ORDER BY t",
            "database": "default",
            "dateColDataType": "DnsDate",
            "dateLoading": false,
            "dateTimeColDataType": "timestamp",
            "datetimeLoading": false,
            "formattedQuery": "<font color=\"darkorange\">SELECT</font> <font color=\"darkcyan\">$timeSeries</font> <font color=\"darkorange\">as</font> t, sumMerge(EdnsCount)<font color=\"yellow\">/</font><font color=\"darkcyan\">$interval</font> <font color=\"darkorange\">as</font> Edns0Present <font color=\"darkorange\">FROM</font> <font color=\"darkcyan\">$table</font> <font color=\"darkorange\">WHERE</font> <font color=\"darkcyan\">$timeFilter</font> <font color=\"yellow\">AND</font> Server <font color=\"darkorange\">IN</font>(<font color=\"darkcyan\">$ServerName</font>)   <font color=\"darkorange\">GROUP BY</font> t <font color=\"darkorange\">ORDER BY</font> t",
            "intervalFactor": 1,
            "query": "SELECT $timeSeries as t, sumMerge(EdnsCount)/$interval as Edns0Present FROM DNS_EDNS WHERE $timeFilter AND Server IN($ServerName)   GROUP BY t ORDER BY t",
            "rawQuery": "SELECT (intDiv(toUInt32(timestamp), 2) * 2) * 1000 as t, sumMerge(EdnsCount)/2 as Edns0Present FROM default.DNS_EDNS WHERE DnsDate >= toDate(1506436944) AND timestamp >= toDateTime(1506436944) AND Server IN('valparaiso')   GROUP BY t ORDER BY t",
            "refId": "A",
            "resultFormat": "time_series",
            "table": "DNS_EDNS",
            "tableLoading": false,
            "tags": [],
            "targetLists": [
              [
                {
                  "params": [
                    "*"
                  ],
                  "type": "field"
                },
                {
                  "params": [],
                  "type": "count"
                }
              ]
            ]
          }
        ],
        "thresholds": [],
        "timeFrom": null,
        "timeShift": null,
        "title": "Edns0 Present in Query",
        "tooltip": {
          "shared": true,
          "sort": 0,
          "value_type": "individual"
        },
        "type": "graph",
        "xaxis": {
          "buckets": null,
          "mode": "time",
          "name": null,
          "show": true,
          "values": []
        },
        "yaxes": [
          {
            "format": "ops",
            "label": null,
            "logBase": 1,
            "max": null,
            "min": null,
            "show": true
          },
          {
            "format": "short",
            "label": null,
            "logBase": 1,
            "max": null,
            "min": null,
            "show": true
          }
        ],
        "yaxis": {
          "align": false,
          "alignLevel": null
        }
      },
      {
        "aliasColors": {},
        "bars": false,
        "dashLength": 10,
        "dashes": false,
        "datasource": "clickhouse",
        "fill": 1,
        "gridPos": {
          "h": 7,
          "w": 12,
          "x": 12,
          "y": 41
        },
        "id": 15,
        "legend": {
          "avg": false,
          "current": false,
          "max": false,
          "min": false,
          "show": true,
          "total": false,
          "values": false
        },
        "lines": true,
        "linewidth": 1,
        "links": [],
        "nullPointMode": "null",
        "percentage": false,
        "pointradius": 5,
        "points": false,
        "renderer": "flot",
        "seriesOverrides": [],
        "spaceLength": 10,
        "stack": false,
        "steppedLine": false,
        "targets": [
          {
            "compiledQuery": "SELECT (intDiv(toUInt32(timestamp), 2) * 2) * 1000 as t, sumMerge(DoBitCount)/2 as DoBitSet FROM DNS_EDNS WHERE DnsDate >= toDate(1506625902) AND timestamp >= toDateTime(1506625902) AND Server IN('default','valparaiso')  GROUP BY t ORDER BY t",
            "database": "default",
            "dateColDataType": "DnsDate",
            "dateLoading": false,
            "dateTimeColDataType": "timestamp",
            "datetimeLoading": false,
            "formattedQuery": "<font color=\"darkorange\">SELECT</font> <font color=\"darkcyan\">$timeSeries</font> <font color=\"darkorange\">as</font> t, sumMerge(DoBitCount)<font color=\"yellow\">/</font><font color=\"darkcyan\">$interval</font> <font color=\"darkorange\">as</font> DoBitSet <font color=\"darkorange\">FROM</font> <font color=\"darkcyan\">$table</font> <font color=\"darkorange\">WHERE</font> <font color=\"darkcyan\">$timeFilter</font> <font color=\"yellow\">AND</font> Server <font color=\"darkorange\">IN</font>(<font color=\"darkcyan\">$ServerName</font>)  <font color=\"darkorange\">GROUP BY</font> t <font color=\"darkorange\">ORDER BY</font> t",
            "intervalFactor": 2,
            "query": "SELECT $timeSeries as t, sumMerge(DoBitCount)/$interval as DoBitSet FROM DNS_EDNS WHERE $timeFilter AND Server IN($ServerName)  GROUP BY t ORDER BY t",
            "rawQuery": "SELECT (intDiv(toUInt32(timestamp), 4) * 4) * 1000 as t, sumMerge(DoBitCount)/4 as DoBitSet FROM default.DNS_EDNS WHERE DnsDate >= toDate(1506436944) AND timestamp >= toDateTime(1506436944) AND Server IN('valparaiso')  GROUP BY t ORDER BY t",
            "refId": "A",
            "resultFormat": "time_series",
            "table": "DNS_EDNS",
            "tableLoading": false,
            "tags": [],
            "targetLists": [
              [
                {
                  "params": [
                    "*"
                  ],
                  "type": "field"
                },
                {
                  "params": [],
                  "type": "count"
                }
              ]
            ]
          }
        ],
        "thresholds": [],
        "timeFrom": null,
        "timeShift": null,
        "title": "DoBit Present in Packet",
        "tooltip": {
          "shared": true,
          "sort": 0,
          "value_type": "individual"
        },
        "type": "graph",
        "xaxis": {
          "buckets": null,
          "mode": "time",
          "name": null,
          "show": true,
          "values": []
        },
        "yaxes": [
          {
            "format": "ops",
            "label": null,
            "logBase": 1,
            "max": null,
            "min": null,
            "show": true
          },
          {
            "format": "short",
            "label": null,
            "logBase": 1,
            "max": null,
            "min": null,
            "show": true
          }
        ],
        "yaxis": {
          "align": false,
          "alignLevel": null
        }
      }
    ],
    "refresh": "30s",
    "schemaVersion": 16,
    "style": "dark",
    "tags": [],
    "templating": {
      "list": [
        {
          "allValue": null,
          "current": {},
          "datasource": "clickhouse",
          "hide": 0,
          "includeAll": true,
          "label": "Server name",
          "multi": true,
          "name": "ServerName",
          "options": [],
          "query": "SELECT DISTINCT Server FROM DNS_LOG",
          "refresh": 1,
          "regex": "",
          "skipUrlSync": false,
          "sort": 1,
          "tagValuesQuery": "SELECT DISTINCT Server FROM DNS_LOG WHERE DnsDate=today()",
          "tags": [
            "Active"
          ],
          "tagsQuery": "SELECT 'Active'",
          "type": "query",
          "useTags": true
        }
      ]
    },
    "time": {
      "from": "now-30m",
      "to": "now"
    },
    "timepicker": {
      "refresh_intervals": [
        "5s",
        "10s",
        "30s",
        "1m",
        "5m",
        "15m",
        "30m",
        "1h",
        "2h",
        "1d"
      ],
      "time_options": [
        "5m",
        "15m",
        "1h",
        "6h",
        "12h",
        "24h",
        "2d",
        "7d",
        "30d"
      ]
    },
    "timezone": "browser",
    "title": "Per Server",
    "uid": "lyxfOy-Wz",
    "version": 2
  }