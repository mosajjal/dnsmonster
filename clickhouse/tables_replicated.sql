CREATE TABLE IF NOT EXISTS DNS_LOG (
  DnsDate Date,
  timestamp DateTime,
  Server String,
  IPVersion UInt8,
  SrcIP UInt64,
  DstIP UInt64,
  Protocol FixedString(3),
  QR UInt8,
  OpCode UInt8,
  Class UInt16,
  Type UInt16,
  Edns0Present UInt8,
  DoBit UInt8,
  FullQuery String,
  ResponseCode UInt8,
  Question String,
  Size UInt16,
  ID UUID
) 
  ENGINE = ReplicatedMergeTree()
  PARTITION BY toYYYYMMDD(DnsDate)
  PRIMARY KEY (timestamp , Server, cityHash64(ID))
  ORDER BY (timestamp, Server, cityHash64(ID))
  SAMPLE BY cityHash64(ID)
  TTL DnsDate + INTERVAL 3000 DAY -- DNS_TTL_VARIABLE
  SETTINGS index_granularity = 8192;

-- View for top queried domains
CREATE MATERIALIZED VIEW IF NOT EXISTS DNS_DOMAIN_COUNT
ENGINE= ReplicatedSummingMergeTree
  PARTITION BY toYYYYMMDD(DnsDate)
  PRIMARY KEY (DnsDate, Server, QH)
  ORDER BY (DnsDate, Server, QH)
  SAMPLE BY QH
  TTL DnsDate + INTERVAL 3000 DAY -- DNS_TTL_VARIABLE
  SETTINGS index_granularity = 8192
  AS SELECT DnsDate, toStartOfMinute(timestamp) as t, Server, Question, cityHash64(Question) as QH, count(*) as c FROM DNS_LOG WHERE QR=0 GROUP BY DnsDate, t, Server, Question;

-- View for unique domain count
CREATE MATERIALIZED VIEW IF NOT EXISTS DNS_DOMAIN_UNIQUE
ENGINE= ReplicatedAggregatingMergeTree
  PARTITION BY toYYYYMMDD(DnsDate)
  PRIMARY KEY (DnsDate, (timestamp, Server)) 
  ORDER BY (DnsDate, (timestamp, Server))
  TTL DnsDate + INTERVAL 3000 DAY -- DNS_TTL_VARIABLE
  SETTINGS index_granularity = 8192
  AS SELECT DnsDate, timestamp, Server, uniqState(Question) AS UniqueDnsCount FROM DNS_LOG WHERE QR=0 GROUP BY Server, DnsDate, timestamp;

-- View for count by protocol
CREATE MATERIALIZED VIEW IF NOT EXISTS DNS_PROTOCOL
ENGINE= ReplicatedSummingMergeTree
  PARTITION BY toYYYYMMDD(DnsDate)
  PRIMARY KEY (DnsDate, Server, PH)
  ORDER BY (DnsDate, Server, PH)
  SAMPLE BY PH
  TTL DnsDate + INTERVAL 3000 DAY -- DNS_TTL_VARIABLE
  SETTINGS index_granularity = 8192
  AS SELECT DnsDate, timestamp, Server, Protocol, cityHash64(Protocol) as PH, count(*) as c FROM DNS_LOG GROUP BY Server, DnsDate, timestamp, Protocol;


-- View with packet sizes
CREATE MATERIALIZED VIEW IF NOT EXISTS DNS_GENERAL_AGGREGATIONS
ENGINE= ReplicatedAggregatingMergeTree
  PARTITION BY toYYYYMMDD(DnsDate)
  PRIMARY KEY (DnsDate, (timestamp, Server)) 
  ORDER BY (DnsDate, (timestamp, Server))
  TTL DnsDate + INTERVAL 3000 DAY -- DNS_TTL_VARIABLE
  SETTINGS index_granularity = 8192
  AS SELECT DnsDate, timestamp, Server, sumState(Size) AS TotalSize, avgState(Size) AS AverageSize FROM DNS_LOG GROUP BY Server, DnsDate, timestamp;


-- View with edns information
CREATE MATERIALIZED VIEW IF NOT EXISTS DNS_EDNS
ENGINE= ReplicatedAggregatingMergeTree
  PARTITION BY toYYYYMMDD(DnsDate)
  PRIMARY KEY (DnsDate, (timestamp, Server)) 
  ORDER BY (DnsDate, (timestamp, Server))
  TTL DnsDate + INTERVAL 3000 DAY -- DNS_TTL_VARIABLE
  SETTINGS index_granularity = 8192
  AS SELECT DnsDate, timestamp, Server, sumState(Edns0Present) as EdnsCount, sumState(DoBit) as DoBitCount FROM DNS_LOG WHERE QR=0 GROUP BY Server, DnsDate, timestamp;


-- View wih query OpCode
CREATE MATERIALIZED VIEW IF NOT EXISTS DNS_OPCODE
ENGINE= ReplicatedSummingMergeTree
  PARTITION BY toYYYYMMDD(DnsDate)
  PRIMARY KEY  (timestamp, Server, OpCode)
  ORDER BY  (timestamp, Server, OpCode)
  SAMPLE BY OpCode
  TTL DnsDate + INTERVAL 3000 DAY -- DNS_TTL_VARIABLE
  SETTINGS index_granularity = 8192
  AS SELECT DnsDate, timestamp, Server, OpCode, count(*) as c FROM DNS_LOG WHERE QR=0 GROUP BY Server, DnsDate, timestamp, OpCode;


-- View with Query Types
CREATE MATERIALIZED VIEW IF NOT EXISTS DNS_TYPE
ENGINE= ReplicatedSummingMergeTree 
  PARTITION BY toYYYYMMDD(DnsDate)
  PRIMARY KEY  (timestamp, Server, Type)
  ORDER BY  (timestamp, Server, Type)
  SAMPLE BY Type
  TTL DnsDate + INTERVAL 3000 DAY -- DNS_TTL_VARIABLE
  SETTINGS index_granularity = 8192
  AS   SELECT DnsDate, timestamp, Server, Type, count(*) as c FROM DNS_LOG WHERE QR=0 GROUP BY Server, DnsDate, timestamp, Type;

-- View with Query Class
CREATE MATERIALIZED VIEW IF NOT EXISTS DNS_CLASS
ENGINE= ReplicatedSummingMergeTree
  PARTITION BY toYYYYMMDD(DnsDate)
  PRIMARY KEY  (timestamp, Server, Class)
  ORDER BY  (timestamp, Server, Class)
  SAMPLE BY Class
  TTL DnsDate + INTERVAL 3000 DAY -- DNS_TTL_VARIABLE
  SETTINGS index_granularity = 8192
  AS SELECT DnsDate, timestamp, Server, Class, count(*) as c FROM DNS_LOG WHERE QR=0 GROUP BY Server, DnsDate, timestamp, Class;  

-- View with query responses
CREATE MATERIALIZED VIEW IF NOT EXISTS DNS_RESPONSECODE
ENGINE= ReplicatedSummingMergeTree
  PARTITION BY toYYYYMMDD(DnsDate)
  PRIMARY KEY  (timestamp, Server, ResponseCode)
  ORDER BY  (timestamp, Server, ResponseCode)
  SAMPLE BY ResponseCode
  TTL DnsDate + INTERVAL 3000 DAY -- DNS_TTL_VARIABLE
  SETTINGS index_granularity = 8192
  AS SELECT DnsDate, timestamp, Server, ResponseCode, count(*) as c FROM DNS_LOG WHERE QR=1 GROUP BY Server, DnsDate, timestamp, ResponseCode;    


-- View with Source IP Prefix
CREATE MATERIALIZED VIEW IF NOT EXISTS DNS_SRCIP_MASK
ENGINE= ReplicatedSummingMergeTree
  PARTITION BY toYYYYMMDD(DnsDate)
  PRIMARY KEY  (timestamp, Server, IPVersion, SrcIP)
  ORDER BY  (timestamp, Server, IPVersion, SrcIP)
  SAMPLE BY SrcIP
  TTL DnsDate + INTERVAL 3000 DAY -- DNS_TTL_VARIABLE
  SETTINGS index_granularity = 8192
  AS SELECT DnsDate, timestamp, Server, IPVersion, SrcIP, count(*) as c FROM DNS_LOG GROUP BY Server, DnsDate, timestamp, IPVersion, SrcIP ;  

-- View with Destination IP Prefix
CREATE MATERIALIZED VIEW IF NOT EXISTS DNS_DSTIP_MASK
ENGINE= ReplicatedSummingMergeTree
  PARTITION BY toYYYYMMDD(DnsDate)
  PRIMARY KEY  (timestamp, Server, IPVersion, DstIP)
  ORDER BY  (timestamp, Server, IPVersion, DstIP)
  SAMPLE BY DstIP
  TTL DnsDate + INTERVAL 3000 DAY -- DNS_TTL_VARIABLE
  SETTINGS index_granularity = 8192
  AS SELECT DnsDate, timestamp, Server, IPVersion, DstIP, count(*) as c FROM DNS_LOG GROUP BY Server, DnsDate, timestamp, IPVersion, DstIP ;  

-- sample queries

-- new domains over the past 24 hours
-- SELECT DISTINCT Question FROM (SELECT Question from DNS_LOG WHERE toStartOfDay(timestamp) > Now() - INTERVAL 1 DAY) AS dns1 LEFT ANTI JOIN (SELECT Question from DNS_LOG WHERE toStartOfDay(timestamp) < Now() - INTERVAL 1 DAY  AND toStartOfDay(timestamp) > (Now() - toIntervalDay(10))  ) as dns2 ON dns1.Question = dns2.Question

-- timeline of request count every 5 minutes
-- SELECT toStartOfFiveMinute(timestamp) as t, count() from DNS_LOG GROUP BY t ORDER BY t

-- 
