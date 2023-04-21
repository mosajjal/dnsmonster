CREATE TABLE IF NOT EXISTS DNS_LOG (
  PacketTime DateTime64,
  IndexTime DateTime64,
  Server LowCardinality(String),
  IPVersion UInt8,
  SrcIP IPv6,
  DstIP IPv6,
  Protocol FixedString(3),
  QR UInt8,
  OpCode UInt8,
  Class UInt16,
  Type UInt16,
  Edns0Present UInt8,
  DoBit UInt8,
  FullQuery String,
  ResponseCode UInt8,
  Question String CODEC(ZSTD(1)),
  Size UInt16
  ) 
  ENGINE = MergeTree()
  PARTITION BY toYYYYMMDD(PacketTime)
  ORDER BY (toStartOfHour(PacketTime), Server,  reverse(Question), toUnixTimestamp(PacketTime))
  SAMPLE BY toUnixTimestamp(PacketTime)
  TTL toDate(PacketTime) + INTERVAL 30 DAY -- DNS_TTL_VARIABLE
  ;

-- View for top queried domains
CREATE TABLE IF NOT EXISTS DNS_DOMAIN_COUNT (
    DnsDate Date,
    t DateTime,
    Server LowCardinality(String),
    Question String CODEC(ZSTD(1)),
    QH UInt64,
    c SimpleAggregateFunction(sum, UInt64)
  )
  ENGINE=AggregatingMergeTree
  PARTITION BY toYYYYMMDD(DnsDate)
  ORDER BY (Server, QH)
  SAMPLE BY QH
  TTL DnsDate + INTERVAL 30 DAY -- DNS_TTL_VARIABLE
  ;

CREATE MATERIALIZED VIEW IF NOT EXISTS DNS_DOMAIN_COUNT_MV TO DNS_DOMAIN_COUNT
  AS SELECT toDate(PacketTime) as DnsDate, toStartOfMinute(PacketTime) as t, Server, Question, cityHash64(Question) as QH, count(*) as c FROM DNS_LOG WHERE QR=0 GROUP BY DnsDate, t, Server, Question;

-- View for unique domain count
CREATE TABLE IF NOT EXISTS DNS_DOMAIN_UNIQUE (
    DnsDate Date,
    timestamp DateTime64,
    Server LowCardinality(String),
    UniqueDnsCount AggregateFunction(uniq, String)
  )
  ENGINE=AggregatingMergeTree
  PARTITION BY toYYYYMMDD(DnsDate)
  ORDER BY (Server, timestamp)
  TTL toDate(timestamp)  + INTERVAL 30 DAY -- DNS_TTL_VARIABLE
  ;

CREATE MATERIALIZED VIEW IF NOT EXISTS DNS_DOMAIN_UNIQUE_MV TO DNS_DOMAIN_UNIQUE
  AS SELECT toDate(PacketTime) as DnsDate, PacketTime as timestamp, Server, uniqState(Question) AS UniqueDnsCount FROM DNS_LOG WHERE QR=0 GROUP BY Server, DnsDate, timestamp;

-- View for count by protocol
CREATE TABLE IF NOT EXISTS DNS_PROTOCOL (
    DnsDate Date,
    timestamp DateTime,
    Server LowCardinality(String),
    Protocol FixedString(3),
    c SimpleAggregateFunction(sum, UInt64)
  )
  ENGINE=AggregatingMergeTree
  PARTITION BY toYYYYMMDD(DnsDate)
  ORDER BY (Server, timestamp)
  TTL DnsDate  + INTERVAL 30 DAY -- DNS_TTL_VARIABLE
  ;

CREATE MATERIALIZED VIEW IF NOT EXISTS DNS_PROTOCOL_MV TO DNS_PROTOCOL
  AS SELECT toDate(PacketTime) as DnsDate, toStartOfMinute(PacketTime) as timestamp, Server, Protocol, count(*) as c FROM DNS_LOG GROUP BY Server, DnsDate, timestamp, Protocol;


-- View with packet sizes
CREATE TABLE IF NOT EXISTS DNS_GENERAL_AGGREGATIONS (
    DnsDate Date,
    timestamp DateTime64,
    Server LowCardinality(String),
    TotalSize AggregateFunction(sum, UInt16),       -- TODO: Change to SimpleAggregateFunction and UInt64
    AverageSize AggregateFunction(avg, UInt16)
  )
  ENGINE=AggregatingMergeTree
  PARTITION BY toYYYYMMDD(DnsDate)
  ORDER BY (Server, timestamp)
  TTL DnsDate  + INTERVAL 30 DAY -- DNS_TTL_VARIABLE
  ;

CREATE MATERIALIZED VIEW IF NOT EXISTS DNS_GENERAL_AGGREGATIONS_MV TO DNS_GENERAL_AGGREGATIONS
  AS SELECT toDate(PacketTime) as DnsDate, PacketTime as timestamp, Server, sumState(Size) AS TotalSize, avgState(Size) AS AverageSize FROM DNS_LOG GROUP BY Server, DnsDate, timestamp;


-- View with edns information
CREATE TABLE IF NOT EXISTS DNS_EDNS (
    DnsDate Date,
    timestamp DateTime64,   -- TODO: This is pretty useless
    Server LowCardinality(String),
    EdnsCount AggregateFunction(sum, UInt8),    -- TODO: These should be SimpleAggregateFunction with UInt64
    DoBitCount AggregateFunction(sum, UInt8),
  )
  ENGINE=AggregatingMergeTree
  PARTITION BY toYYYYMMDD(DnsDate)
  ORDER BY (Server, timestamp)
  TTL DnsDate  + INTERVAL 30 DAY -- DNS_TTL_VARIABLE
  ;

CREATE MATERIALIZED VIEW IF NOT EXISTS DNS_EDNS_MV TO DNS_EDNS
  AS SELECT toDate(PacketTime) as DnsDate, PacketTime as timestamp, Server, sumState(Edns0Present) as EdnsCount, sumState(DoBit) as DoBitCount FROM DNS_LOG WHERE QR=0 GROUP BY Server, DnsDate, timestamp;


-- View wih query OpCode
CREATE TABLE IF NOT EXISTS DNS_OPCODE (
    DnsDate Date,
    timestamp DateTime64,
    Server LowCardinality(String),
    OpCode UInt8,
    c SimpleAggregateFunction(sum, UInt64)
  )
  ENGINE=AggregatingMergeTree
  PARTITION BY toYYYYMMDD(DnsDate)
  ORDER BY (Server, OpCode, timestamp)
  SAMPLE BY OpCode
  TTL DnsDate + INTERVAL 30 DAY -- DNS_TTL_VARIABLE
  ;

CREATE MATERIALIZED VIEW IF NOT EXISTS DNS_OPCODE_MV TO DNS_OPCODE
  AS SELECT toDate(PacketTime) as DnsDate, PacketTime as timestamp, Server, OpCode, count(*) as c FROM DNS_LOG WHERE QR=0 GROUP BY Server, DnsDate, timestamp, OpCode;


-- View with Query Types
CREATE TABLE IF NOT EXISTS DNS_TYPE (
    DnsDate Date,
    timestamp DateTime,
    Server LowCardinality(String),
    Type UInt16,
    c SimpleAggregateFunction(sum, UInt64)
  )
  ENGINE=AggregatingMergeTree
  PARTITION BY toYYYYMMDD(DnsDate)
  ORDER BY (Server, Type, timestamp)
  SAMPLE BY Type
  TTL DnsDate  + INTERVAL 30 DAY -- DNS_TTL_VARIABLE
  ;

CREATE MATERIALIZED VIEW IF NOT EXISTS DNS_TYPE_MV TO DNS_TYPE
  AS SELECT toDate(PacketTime) as DnsDate, PacketTime as timestamp, Server, Type, count(*) as c FROM DNS_LOG WHERE QR=0 GROUP BY Server, DnsDate, timestamp, Type;

-- View with Query Class
CREATE TABLE IF NOT EXISTS DNS_CLASS (
    DnsDate Date,
    timestamp DateTime,
    Server LowCardinality(String),
    Class UInt16,
    c SimpleAggregateFunction(sum, UInt64)
  )
  ENGINE=AggregatingMergeTree
  PARTITION BY toYYYYMMDD(DnsDate)
  ORDER BY (Server, Class, timestamp)
  SAMPLE BY Class
  TTL DnsDate  + INTERVAL 30 DAY -- DNS_TTL_VARIABLE
  ;
CREATE MATERIALIZED VIEW IF NOT EXISTS DNS_CLASS_MV TO DNS_CLASS
  AS SELECT toDate(PacketTime) as DnsDate, PacketTime as timestamp, Server, Class, count(*) as c FROM DNS_LOG WHERE QR=0 GROUP BY Server, DnsDate, timestamp, Class;  

-- View with query responses
CREATE TABLE IF NOT EXISTS DNS_RESPONSECODE (
    DnsDate Date,
    timestamp DateTime,
    Server LowCardinality(String),
    ResponseCode UInt8,
    c SimpleAggregateFunction(sum, UInt64)
  )
  ENGINE=AggregatingMergeTree
  PARTITION BY toYYYYMMDD(DnsDate)
  ORDER BY (Server, ResponseCode, timestamp)
  SAMPLE BY ResponseCode
  TTL DnsDate  + INTERVAL 30 DAY -- DNS_TTL_VARIABLE
  ;
CREATE MATERIALIZED VIEW IF NOT EXISTS DNS_RESPONSECODE_MV TO DNS_RESPONSECODE
  AS SELECT toDate(PacketTime) as DnsDate, PacketTime as timestamp, Server, ResponseCode, count(*) as c FROM DNS_LOG WHERE QR=1 GROUP BY Server, DnsDate, timestamp, ResponseCode;    


-- View with Source IP Prefix
CREATE TABLE IF NOT EXISTS DNS_SRCIP_MASK (
    DnsDate Date,
    timestamp DateTime,
    Server LowCardinality(String),
    IPVersion UInt8,
    SrcIP IPv6,
    c SimpleAggregateFunction(sum, UInt64)
  )
  ENGINE=AggregatingMergeTree
  PARTITION BY toYYYYMMDD(DnsDate)
  ORDER BY (Server, IPVersion, cityHash64(SrcIP))
  SAMPLE BY cityHash64(SrcIP)
  TTL DnsDate + INTERVAL 30 DAY -- DNS_TTL_VARIABLE
  ;
CREATE MATERIALIZED VIEW IF NOT EXISTS DNS_SRCIP_MASK_MV TO DNS_SRCIP_MASK
  AS SELECT toDate(PacketTime) as DnsDate, PacketTime as timestamp, Server, IPVersion, SrcIP, count(*) as c FROM DNS_LOG GROUP BY Server, DnsDate, timestamp, IPVersion, SrcIP ;  

-- View with Destination IP Prefix
CREATE TABLE IF NOT EXISTS DNS_DSTIP_MASK (
    DnsDate Date,
    timestamp DateTime,
    Server LowCardinality(String),
    IPVersion UInt8,
    DstIP IPv6,
    c SimpleAggregateFunction(sum, UInt64)
  )
  ENGINE=AggregatingMergeTree
  PARTITION BY toYYYYMMDD(DnsDate)
  ORDER BY (Server, IPVersion, cityHash64(DstIP))
  SAMPLE BY cityHash64(DstIP)
  TTL DnsDate + INTERVAL 30 DAY -- DNS_TTL_VARIABLE
  ;
CREATE MATERIALIZED VIEW IF NOT EXISTS DNS_DSTIP_MASK_MV TO DNS_DSTIP_MASK
  AS SELECT toDate(PacketTime) as DnsDate, PacketTime as timestamp, Server, IPVersion, DstIP, count(*) as c FROM DNS_LOG GROUP BY Server, DnsDate, timestamp, IPVersion, DstIP ;  

-- sample queries

-- new domains over the past 24 hours
-- SELECT DISTINCT Question FROM (SELECT Question from DNS_LOG WHERE toStartOfDay(timestamp) > Now() - INTERVAL 1 DAY) AS dns1 LEFT ANTI JOIN (SELECT Question from DNS_LOG WHERE toStartOfDay(timestamp) < Now() - INTERVAL 1 DAY  AND toStartOfDay(timestamp) > (Now() - toIntervalDay(10))  ) as dns2 ON dns1.Question = dns2.Question

-- timeline of request count every 5 minutes
-- SELECT toStartOfFiveMinute(timestamp) as t, count() from DNS_LOG GROUP BY t ORDER BY t

-- 
