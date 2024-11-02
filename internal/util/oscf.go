package util

import (
	json "github.com/bytedance/sonic"
	"github.com/miekg/dns"
)

// OCSFDNSActivity represents DNS activity in OCSF format
type OCSFDNSActivity struct {
	TypeUID     int   `json:"type_uid"`
	CategoryUID int   `json:"category_uid"`
	ClassUID    int   `json:"class_uid"`
	Time        int64 `json:"time"`
	ActivityID  int   `json:"activity_id"`
	SeverityID  int   `json:"severity_id"`

	Query   *OCSFDNSQuery   `json:"query,omitempty"`
	Answers []OCSFDNSAnswer `json:"answers,omitempty"`

	RCode   string `json:"rcode,omitempty"`
	RCodeID int    `json:"rcode_id,omitempty"`

	// Network endpoints
	SrcEndpoint *OCSFNetworkEndpoint `json:"src_endpoint,omitempty"`
	DstEndpoint *OCSFNetworkEndpoint `json:"dst_endpoint,omitempty"`

	Metadata struct {
		Product struct {
			Name       string `json:"name"`
			VendorName string `json:"vendor_name"`
		} `json:"product"`
		Version string `json:"version"`
	} `json:"metadata"`
}

// OCSFNetworkEndpoint represents a network endpoint in OCSF format
type OCSFNetworkEndpoint struct {
	IP       string `json:"ip,omitempty"`
	Port     int    `json:"port,omitempty"`
	Hostname string `json:"hostname,omitempty"`
	Domain   string `json:"domain,omitempty"`
	Protocol string `json:"protocol,omitempty"`
}

// OCSFDNSQuery matches OCSF DNS query schema
type OCSFDNSQuery struct {
	Class     string `json:"class"`      // Resource Record Class
	Hostname  string `json:"hostname"`   // Query hostname
	Opcode    string `json:"opcode"`     // DNS opcode
	OpcodeID  int    `json:"opcode_id"`  // DNS opcode ID
	PacketUID int    `json:"packet_uid"` // Packet identifier
	Type      string `json:"type"`       // Resource Record Type
}

// OCSFDNSAnswer matches OCSF DNS answer schema
type OCSFDNSAnswer struct {
	Class     string   `json:"class"`      // Resource Record Class
	FlagIDs   []int    `json:"flag_ids"`   // DNS Header Flag IDs
	Flags     []string `json:"flags"`      // DNS Header Flag names
	PacketUID int      `json:"packet_uid"` // Packet identifier
	RData     string   `json:"rdata"`      // DNS Resource Record data
	TTL       int      `json:"ttl"`        // Time to live
	Type      string   `json:"type"`       // Resource Record Type
}

// ToOCSF converts DNSResult to OCSF format
func ToOCSF(result *DNSResult) *OCSFDNSActivity {
	activity := &OCSFDNSActivity{
		TypeUID:     4003, // DNS Activity
		CategoryUID: 4,    // Network Activity
		ClassUID:    4003, // DNS Activity
		Time:        result.Timestamp.Unix(),
		ActivityID:  1, // Query
		SeverityID:  0, // Info
	}

	// Set metadata
	activity.Metadata.Product.Name = "dns"
	activity.Metadata.Product.VendorName = "dnsmonster"
	activity.Metadata.Version = "1.0.0"

	// Convert query
	if len(result.DNS.Question) > 0 {
		q := result.DNS.Question[0]
		activity.Query = &OCSFDNSQuery{
			Hostname:  q.Name,
			Type:      dns.TypeToString[q.Qtype],
			Class:     dns.ClassToString[q.Qclass],
			Opcode:    dns.OpcodeToString[result.DNS.Opcode],
			OpcodeID:  int(result.DNS.Opcode),
			PacketUID: int(result.DNS.Id),
		}
	}

	// Convert answers
	activity.Answers = make([]OCSFDNSAnswer, len(result.DNS.Answer))
	for i, a := range result.DNS.Answer {
		hdr := a.Header()
		activity.Answers[i] = OCSFDNSAnswer{
			RData:     a.String(),
			Type:      dns.TypeToString[hdr.Rrtype],
			Class:     dns.ClassToString[hdr.Class],
			TTL:       int(hdr.Ttl),
			PacketUID: int(result.DNS.Id),
			Flags:     getFlagsList(result.DNS),
			FlagIDs:   getFlagIDs(result.DNS),
		}
	}

	// Set response code
	activity.RCode = dns.RcodeToString[result.DNS.Rcode]
	activity.RCodeID = result.DNS.Rcode

	// Add network endpoints
	activity.SrcEndpoint = &OCSFNetworkEndpoint{
		IP:       result.SrcIP.String(),
		Port:     int(result.SrcPort),
		Protocol: result.Protocol,
	}

	activity.DstEndpoint = &OCSFNetworkEndpoint{
		IP:       result.DstIP.String(),
		Port:     int(result.DstPort),
		Protocol: result.Protocol,
	}

	return activity
}

// Helper functions for flags
func getFlagsList(msg dns.Msg) []string {
	var flags []string
	if msg.Response {
		flags = append(flags, "Response")
	}
	if msg.Authoritative {
		flags = append(flags, "Authoritative")
	}
	if msg.Truncated {
		flags = append(flags, "Truncated")
	}
	if msg.RecursionDesired {
		flags = append(flags, "RecursionDesired")
	}
	if msg.RecursionAvailable {
		flags = append(flags, "RecursionAvailable")
	}
	if msg.AuthenticatedData {
		flags = append(flags, "AuthenticatedData")
	}
	if msg.CheckingDisabled {
		flags = append(flags, "CheckingDisabled")
	}
	return flags
}

func getFlagIDs(msg dns.Msg) []int {
	var flagIDs []int
	if msg.Response {
		flagIDs = append(flagIDs, 0)
	}
	if msg.Authoritative {
		flagIDs = append(flagIDs, 1)
	}
	if msg.Truncated {
		flagIDs = append(flagIDs, 2)
	}
	if msg.RecursionDesired {
		flagIDs = append(flagIDs, 3)
	}
	if msg.RecursionAvailable {
		flagIDs = append(flagIDs, 4)
	}
	if msg.AuthenticatedData {
		flagIDs = append(flagIDs, 5)
	}
	if msg.CheckingDisabled {
		flagIDs = append(flagIDs, 6)
	}
	return flagIDs
}

// Convert OCSF format back to DNS message
func FromOCSF(activity *OCSFDNSActivity) *dns.Msg {
	msg := &dns.Msg{}

	// Convert query
	if activity.Query != nil {
		msg.Question = make([]dns.Question, 1)
		msg.Question[0] = dns.Question{
			Name:   activity.Query.Hostname,
			Qtype:  dns.StringToType[activity.Query.Type],
			Qclass: dns.StringToClass[activity.Query.Class],
		}
	}

	// Convert answers
	msg.Answer = make([]dns.RR, len(activity.Answers))
	for i, a := range activity.Answers {
		rr, err := dns.NewRR(a.RData)
		if err != nil {
			continue
		}
		msg.Answer[i] = rr
	}

	// Set response code
	msg.Rcode = activity.RCodeID

	return msg
}

// Convert OCSF structure to native Go map for Avro encoding
func (a *OCSFDNSActivity) toNativeMap() map[string]interface{} {
	native := map[string]interface{}{
		"type_uid":     a.TypeUID,
		"category_uid": a.CategoryUID,
		"class_uid":    a.ClassUID,
		"time":         a.Time,
		"activity_id":  a.ActivityID,
		"severity_id":  a.SeverityID,
		"rcode":        a.RCode,
		"rcode_id":     a.RCodeID,
		"metadata": map[string]interface{}{
			"product": map[string]interface{}{
				"name":        a.Metadata.Product.Name,
				"vendor_name": a.Metadata.Product.VendorName,
			},
			"version": a.Metadata.Version,
		},
	}

	// Convert query if exists
	if a.Query != nil {
		native["query"] = map[string]interface{}{
			"class":      a.Query.Class,
			"hostname":   a.Query.Hostname,
			"opcode":     a.Query.Opcode,
			"opcode_id":  a.Query.OpcodeID,
			"packet_uid": a.Query.PacketUID,
			"type":       a.Query.Type,
		}
	}

	// Convert answers
	answers := make([]map[string]interface{}, len(a.Answers))
	for i, ans := range a.Answers {
		answers[i] = map[string]interface{}{
			"class":      ans.Class,
			"flag_ids":   ans.FlagIDs,
			"flags":      ans.Flags,
			"packet_uid": ans.PacketUID,
			"rdata":      ans.RData,
			"ttl":        ans.TTL,
			"type":       ans.Type,
		}
	}
	native["answers"] = answers

	return native
}

// Convert native Go map back to OCSF structure
func (a *OCSFDNSActivity) fromNativeMap(native map[string]interface{}) {
	// Convert basic fields
	if v, ok := native["type_uid"].(int32); ok {
		a.TypeUID = int(v)
	}
	if v, ok := native["category_uid"].(int32); ok {
		a.CategoryUID = int(v)
	}
	if v, ok := native["class_uid"].(int32); ok {
		a.ClassUID = int(v)
	}
	if v, ok := native["time"].(int64); ok {
		a.Time = v
	}
	if v, ok := native["activity_id"].(int32); ok {
		a.ActivityID = int(v)
	}
	if v, ok := native["severity_id"].(int32); ok {
		a.SeverityID = int(v)
	}
	if v, ok := native["rcode"].(string); ok {
		a.RCode = v
	}
	if v, ok := native["rcode_id"].(int32); ok {
		a.RCodeID = int(v)
	}

	// Convert metadata
	if meta, ok := native["metadata"].(map[string]interface{}); ok {
		if prod, ok := meta["product"].(map[string]interface{}); ok {
			if name, ok := prod["name"].(string); ok {
				a.Metadata.Product.Name = name
			}
			if vendor, ok := prod["vendor_name"].(string); ok {
				a.Metadata.Product.VendorName = vendor
			}
		}
		if version, ok := meta["version"].(string); ok {
			a.Metadata.Version = version
		}
	}

	// Convert query
	if q, ok := native["query"].(map[string]interface{}); ok {
		query := &OCSFDNSQuery{}
		if v, ok := q["class"].(string); ok {
			query.Class = v
		}
		if v, ok := q["hostname"].(string); ok {
			query.Hostname = v
		}
		if v, ok := q["opcode"].(string); ok {
			query.Opcode = v
		}
		if v, ok := q["opcode_id"].(int32); ok {
			query.OpcodeID = int(v)
		}
		if v, ok := q["packet_uid"].(int32); ok {
			query.PacketUID = int(v)
		}
		if v, ok := q["type"].(string); ok {
			query.Type = v
		}
		a.Query = query
	}

	// Convert answers
	if ans, ok := native["answers"].([]interface{}); ok {
		a.Answers = make([]OCSFDNSAnswer, len(ans))
		for i, v := range ans {
			if answer, ok := v.(map[string]interface{}); ok {
				if class, ok := answer["class"].(string); ok {
					a.Answers[i].Class = class
				}
				if flagIDs, ok := answer["flag_ids"].([]interface{}); ok {
					ids := make([]int, len(flagIDs))
					for j, id := range flagIDs {
						if val, ok := id.(int32); ok {
							ids[j] = int(val)
						}
					}
					a.Answers[i].FlagIDs = ids
				}
				if flags, ok := answer["flags"].([]interface{}); ok {
					strs := make([]string, len(flags))
					for j, f := range flags {
						if val, ok := f.(string); ok {
							strs[j] = val
						}
					}
					a.Answers[i].Flags = strs
				}
				if v, ok := answer["packet_uid"].(int32); ok {
					a.Answers[i].PacketUID = int(v)
				}
				if v, ok := answer["rdata"].(string); ok {
					a.Answers[i].RData = v
				}
				if v, ok := answer["ttl"].(int32); ok {
					a.Answers[i].TTL = int(v)
				}
				if v, ok := answer["type"].(string); ok {
					a.Answers[i].Type = v
				}
			}
		}
	}
}

// OCSFMarshaler implements OCSF JSON marshaling
type OCSFMarshaler struct{}

func (m OCSFMarshaler) Marshal(d DNSResult) []byte {
	activity := ToOCSF(&d)
	j, err := json.Marshal(activity)
	if err != nil {
		return nil
	}
	return j
}

func (m OCSFMarshaler) Init() (string, error) {
	return "", nil
}

// OCSFUnmarshaler implements OCSF JSON unmarshaling
type OCSFUnmarshaler struct{}

func (u OCSFUnmarshaler) Unmarshal(data []byte) (*dns.Msg, error) {
	activity := &OCSFDNSActivity{}
	if err := json.Unmarshal(data, activity); err != nil {
		return nil, err
	}
	return FromOCSF(activity), nil
}
