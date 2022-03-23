//go:build !linux || !amd64 || go1.18
// +build !linux !amd64 go1.18

package util

import "encoding/json"

func (d *DNSResult) GetJson() string {
	res, _ := json.Marshal(d)
	return string(res)
}
