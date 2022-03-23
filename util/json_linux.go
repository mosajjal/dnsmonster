//go:build linux && amd64 && !go1.18
// +build linux,amd64,!go1.18

package util

import (
	"github.com/bytedance/sonic"
)

func (d *DNSResult) GetJson() string {
	res, _ := sonic.Marshal(d)
	return string(res)
}
