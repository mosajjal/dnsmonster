CREATE TABLE IF NOT EXISTS DNS_LOG (
  DnsDate Date,
  timestamp DateTime,
  Server String,
  IPVersion UInt8,
  IPPrefix UInt32,
  Protocol FixedString(3),
  QR UInt8,
  OpCode UInt8,
  Class UInt16,
  Type UInt16,
  Edns0Present UInt8,
  DoBit UInt8,
  ResponseCode UInt8,
  Question String,
  Size UInt16,
  ID UUID
) 
  ENGINE = MergeTree()
  PARTITION BY toYYYYMMDD(DnsDate)
  PRIMARY KEY (timestamp , Server, cityHash64(ID))
  ORDER BY (timestamp, Server, cityHash64(ID))
  SAMPLE BY cityHash64(ID)
  TTL DnsDate + INTERVAL 30 DAY -- DNS_TTL_VARIABLE
  SETTINGS index_granularity = 8192;

-- View for top queried domains
CREATE MATERIALIZED VIEW IF NOT EXISTS DNS_DOMAIN_COUNT
ENGINE=SummingMergeTree(DnsDate, (t, Server, Question), 8192, c) AS
  SELECT DnsDate, toStartOfMinute(timestamp) as t, Server, Question, count(*) as c FROM DNS_LOG WHERE QR=0 GROUP BY DnsDate, t, Server, Question;
ALTER TABLE `.inner.DNS_DOMAIN_COUNT` MODIFY
 TTL DnsDate + INTERVAL 30 DAY -- DNS_TTL_VARIABLE
;

-- View for unique domain count
CREATE MATERIALIZED VIEW IF NOT EXISTS DNS_DOMAIN_UNIQUE
ENGINE=AggregatingMergeTree(DnsDate, (timestamp, Server), 8192) AS
  SELECT DnsDate, timestamp, Server, uniqState(Question) AS UniqueDnsCount FROM DNS_LOG WHERE QR=0 GROUP BY Server, DnsDate, timestamp;
ALTER TABLE `.inner.DNS_DOMAIN_UNIQUE` MODIFY
 TTL DnsDate + INTERVAL 30 DAY -- DNS_TTL_VARIABLE
;

-- View for count by protocol
CREATE MATERIALIZED VIEW IF NOT EXISTS DNS_PROTOCOL
ENGINE=SummingMergeTree(DnsDate, (timestamp, Server, Protocol), 8192, (c)) AS
  SELECT DnsDate, timestamp, Server, Protocol, count(*) as c FROM DNS_LOG GROUP BY Server, DnsDate, timestamp, Protocol;
ALTER TABLE `.inner.DNS_PROTOCOL` MODIFY
 TTL DnsDate + INTERVAL 30 DAY -- DNS_TTL_VARIABLE
;

-- View with packet sizes
CREATE MATERIALIZED VIEW IF NOT EXISTS DNS_GENERAL_AGGREGATIONS
ENGINE=AggregatingMergeTree(DnsDate, (timestamp, Server), 8192) AS
SELECT DnsDate, timestamp, Server, sumState(Size) AS TotalSize, avgState(Size) AS AverageSize FROM DNS_LOG GROUP BY Server, DnsDate, timestamp;
ALTER TABLE `.inner.DNS_GENERAL_AGGREGATIONS` MODIFY 
 TTL DnsDate + INTERVAL 30 DAY -- DNS_TTL_VARIABLE
;

-- View with edns information
CREATE MATERIALIZED VIEW IF NOT EXISTS DNS_EDNS
ENGINE=AggregatingMergeTree(DnsDate, (timestamp, Server), 8192) AS
  SELECT DnsDate, timestamp, Server, sumState(Edns0Present) as EdnsCount, sumState(DoBit) as DoBitCount FROM DNS_LOG WHERE QR=0 GROUP BY Server, DnsDate, timestamp;
ALTER TABLE `.inner.DNS_EDNS` MODIFY
 TTL DnsDate + INTERVAL 30 DAY -- DNS_TTL_VARIABLE
;

-- View wih query OpCode
CREATE MATERIALIZED VIEW IF NOT EXISTS DNS_OPCODE
ENGINE=SummingMergeTree(DnsDate, (timestamp, Server, OpCode), 8192, c) AS
  SELECT DnsDate, timestamp, Server, OpCode, count(*) as c FROM DNS_LOG WHERE QR=0 GROUP BY Server, DnsDate, timestamp, OpCode;
ALTER TABLE `.inner.DNS_OPCODE` MODIFY
 TTL DnsDate + INTERVAL 30 DAY -- DNS_TTL_VARIABLE
;

-- View with Query Types
CREATE MATERIALIZED VIEW IF NOT EXISTS DNS_TYPE
ENGINE=SummingMergeTree(DnsDate, (timestamp, Server, Type), 8192, c) AS
  SELECT DnsDate, timestamp, Server, Type, count(*) as c FROM DNS_LOG WHERE QR=0 GROUP BY Server, DnsDate, timestamp, Type;
ALTER TABLE `.inner.DNS_TYPE` MODIFY
 TTL DnsDate + INTERVAL 30 DAY -- DNS_TTL_VARIABLE
;

-- View with Query Class
CREATE MATERIALIZED VIEW IF NOT EXISTS DNS_CLASS
ENGINE=SummingMergeTree(DnsDate, (timestamp, Server, Class), 8192, c) AS
  SELECT DnsDate, timestamp, Server, Class, count(*) as c FROM DNS_LOG WHERE QR=0 GROUP BY Server, DnsDate, timestamp, Class;
ALTER TABLE `.inner.DNS_CLASS` MODIFY
 TTL DnsDate + INTERVAL 30 DAY -- DNS_TTL_VARIABLE
;

-- View with query responses
CREATE MATERIALIZED VIEW IF NOT EXISTS DNS_RESPONSECODE
ENGINE=SummingMergeTree(DnsDate, (timestamp, Server, ResponseCode), 8192, c) AS
  SELECT DnsDate, timestamp, Server, ResponseCode, count(*) as c FROM DNS_LOG WHERE QR=1 GROUP BY Server, DnsDate, timestamp, ResponseCode;
ALTER TABLE `.inner.DNS_RESPONSECODE` MODIFY
 TTL DnsDate + INTERVAL 30 DAY -- DNS_TTL_VARIABLE
;

-- View with IP Prefix
CREATE MATERIALIZED VIEW IF NOT EXISTS DNS_IP_MASK
ENGINE=SummingMergeTree(DnsDate, (timestamp, Server, IPVersion, IPPrefix), 8192, c) AS
  SELECT DnsDate, timestamp, Server, IPVersion, IPPrefix, count(*) as c FROM DNS_LOG GROUP BY Server, DnsDate, timestamp, IPVersion, IPPrefix;
  ALTER TABLE `.inner.DNS_IP_MASK` MODIFY
   TTL DnsDate + INTERVAL 30 DAY -- DNS_TTL_VARIABLE
;
