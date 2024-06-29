---
title: "GeoIP Options"
linkTitle: "GeoIP options"
weight: 4
---

Let's go through some examples of how to set up `dnsmonster` geoip

### How to use GeoIP lookup

To utilize this feature, you must possess at least one of the MaxMind databases: Country, City, or ASN. Please register and download the necessary MaxMind GeoIP databases from their official website: [MaxMind](https://www.maxmind.com/en/).

For example:
```
dnsmonster --geoip --geoipcountryfile GeoLite2-Country.mmdb --geoipcityfile GeoLite2-City.mmdb --geoipasnfile GeoLite2-ASN.mmdb
```

This feature modify the output by including GeoIP fields, as following below:

```
{
  "Timestamp": "2024-06-04T17:28:00.000305Z",
  "DNS": {
    "Id": 64187,
    "Response": true,
    "Opcode": 0,
    "Authoritative": false,
    "Truncated": false,
    "RecursionDesired": false,
    "RecursionAvailable": false,
    "Zero": false,
    "AuthenticatedData": false,
    "CheckingDisabled": true,
    "Rcode": 5,
    "Question": [
      {
        "Name": "example.com.",
        "Qtype": 5,
        "Qclass": 1
      }
    ],
    "Answer": null,
    "Ns": null,
    "Extra": [
      {
        "Hdr": {
          "Name": ".",
          "Rrtype": 41,
          "Class": 1220,
          "Ttl": 32768,
          "Rdlength": 0
        },
        "Option": null
      }
    ]
  },
  "IPVersion": 4,
  "SrcIP": "8.8.8.8",
  "DstIP": "46.5.124.5",
  "Protocol": "udp",
  "PacketLength": 55,
  "GeoIP": {
    "Continent": "NA",
    "CountryISOCode": "US",
    "City": "Mountain View",
    "ASN": "15169",
    "ASO": "Google LLC"
  }
}
```
