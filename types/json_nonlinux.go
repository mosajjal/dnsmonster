//go:build !linux || !amd64
// +build !linux !amd64

package types

import "encoding/json"

func (d *DNSResult) String() string {
	res, _ := json.Marshal(d)
	return string(res)
}
