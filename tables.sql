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
  ResponceCode UInt8,
  Question String,
  Size UInt16
) engine=MergeTree(DnsDate, (timestamp, Server), 8192);

-- View for top queried domains
CREATE MATERIALIZED VIEW IF NOT EXISTS DNS_DOMAIN_COUNT
ENGINE=SummingMergeTree(DnsDate, (t, Server, Question), 8192, c) AS
  SELECT DnsDate, toStartOfMinute(timestamp) as t, Server, Question, count(*) as c FROM DNS_LOG WHERE QR=0 GROUP BY DnsDate, t, Server, Question;

-- View for unique domain count
CREATE MATERIALIZED VIEW IF NOT EXISTS DNS_DOMAIN_UNIQUE
ENGINE=AggregatingMergeTree(DnsDate, (timestamp, Server), 8192) AS
  SELECT DnsDate, timestamp, Server, uniqState(Question) AS UniqueDnsCount FROM DNS_LOG WHERE QR=0 GROUP BY Server, DnsDate, timestamp;

-- View for count by protocol
CREATE MATERIALIZED VIEW IF NOT EXISTS DNS_PROTOCOL
ENGINE=SummingMergeTree(DnsDate, (timestamp, Server, Protocol), 8192, (c)) AS
  SELECT DnsDate, timestamp, Server, Protocol, count(*) as c FROM DNS_LOG GROUP BY Server, DnsDate, timestamp, Protocol;

-- View with packet sizes
CREATE MATERIALIZED VIEW IF NOT EXISTS DNS_GENERAL_AGGREGATIONS
ENGINE=AggregatingMergeTree(DnsDate, (timestamp, Server), 8192) AS
SELECT DnsDate, timestamp, Server, sumState(Size) AS TotalSize, avgState(Size) AS AverageSize FROM DNS_LOG GROUP BY Server, DnsDate, timestamp;

-- View with edns information
CREATE MATERIALIZED VIEW IF NOT EXISTS DNS_EDNS
ENGINE=AggregatingMergeTree(DnsDate, (timestamp, Server), 8192) AS
  SELECT DnsDate, timestamp, Server, sumState(Edns0Present) as EdnsCount, sumState(DoBit) as DoBitCount FROM DNS_LOG WHERE QR=0 GROUP BY Server, DnsDate, timestamp;

-- View wih query OpCode
CREATE MATERIALIZED VIEW IF NOT EXISTS DNS_OPCODE
ENGINE=SummingMergeTree(DnsDate, (timestamp, Server, OpCode), 8192, c) AS
  SELECT DnsDate, timestamp, Server, OpCode, count(*) as c FROM DNS_LOG WHERE QR=0 GROUP BY Server, DnsDate, timestamp, OpCode;

-- View with Query Types
CREATE MATERIALIZED VIEW IF NOT EXISTS DNS_TYPE
ENGINE=SummingMergeTree(DnsDate, (timestamp, Server, Type), 8192, c) AS
  SELECT DnsDate, timestamp, Server, Type, count(*) as c FROM DNS_LOG WHERE QR=0 GROUP BY Server, DnsDate, timestamp, Type;

-- View with Query Class
CREATE MATERIALIZED VIEW IF NOT EXISTS DNS_CLASS
ENGINE=SummingMergeTree(DnsDate, (timestamp, Server, Class), 8192, c) AS
  SELECT DnsDate, timestamp, Server, Class, count(*) as c FROM DNS_LOG WHERE QR=0 GROUP BY Server, DnsDate, timestamp, Class;

-- View with query responses
CREATE MATERIALIZED VIEW IF NOT EXISTS DNS_RESPONCECODE
ENGINE=SummingMergeTree(DnsDate, (timestamp, Server, ResponceCode), 8192, c) AS
  SELECT DnsDate, timestamp, Server, ResponceCode, count(*) as c FROM DNS_LOG WHERE QR=1 GROUP BY Server, DnsDate, timestamp, ResponceCode;

-- View with IP Prefix
CREATE MATERIALIZED VIEW IF NOT EXISTS DNS_IP_MASK
ENGINE=SummingMergeTree(DnsDate, (timestamp, Server, IPVersion, IPPrefix), 8192, c) AS
  SELECT DnsDate, timestamp, Server, IPVersion, IPPrefix, count(*) as c FROM DNS_LOG GROUP BY Server, DnsDate, timestamp, IPVersion, IPPrefix;
