//go:build linux && amd64
// +build linux,amd64

package util

import (
	"github.com/bytedance/sonic"
)

func (d *DNSResult) GetJson() string {
	res, _ := sonic.Marshal(d)
	return string(res)
}
