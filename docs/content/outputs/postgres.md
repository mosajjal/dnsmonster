+++
title = "PostgreSQL"
description = "Experimental dnsmonster output for PostgreSQL and compatible engines such as CockroachDB."
weight = 7
+++

PostgreSQL is regarded as the world's most advanced open source database. `dnsmonster` has
experimental support for writing to PostgreSQL and to compatible engines such as CockroachDB.

## Configuration parameters

```sh
# What should be written to Psql. options:
#	0: Disable Output
#	1: Enable Output without any filters
#	2: Enable Output and apply skipdomains logic
#	3: Enable Output and apply allowdomains logic
#	4: Enable Output and apply both skip and allow domains logic
--psqlOutputType=0

# Psql endpoint used. must be in uri format. example: postgres://username:password@hostname:port/database?sslmode=disable
--psqlEndpoint=

# Number of PSQL workers
--psqlWorkers=1

# Psql Batch Size
--psqlBatchSize=1

# Interval between sending results to Psql if Batch size is not filled. Any value larger than zero takes precedence over Batch Size
--psqlBatchDelay=0s

# Timeout for any INSERT operation before we consider them failed
--psqlBatchTimeout=5s

# Save full packet query and response in JSON format.
--psqlSaveFullQuery
```
