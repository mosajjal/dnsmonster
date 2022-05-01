//go:build !linux || !amd64
// +build !linux !amd64

package util

import "encoding/json"

type JsonOutput struct{}

func (j JsonOutput) Marshal(d DNSResult) string {
	res, _ := json.Marshal(d)
	return string(res)
}

func (j JsonOutput) Init() (string, error) {
	return "", nil
}
