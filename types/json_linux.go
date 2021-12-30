//go:build linux && amd64
// +build linux,amd64

package types

import (
	"github.com/bytedance/sonic"
)

func (d *DNSResult) String() string {
	res, _ := sonic.Marshal(d)
	return string(res)
}
