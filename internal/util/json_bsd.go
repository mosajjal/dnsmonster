//go:build openbsd || freebsd || dragonfly || netbsd

package util

import (
	"encoding/json"
)

type jsonOutput struct{}

func (j jsonOutput) Marshal(d DNSResult) string {
	res, _ := json.Marshal(d)
	return string(res)
}

func (j jsonOutput) Init() (string, error) {
	return "", nil
}
