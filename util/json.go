//go:build !(openbsd || freebsd || dragonfly || netbsd)

package util

import (
	"github.com/bytedance/sonic"
)

type jsonOutput struct{}

func (j jsonOutput) Marshal(d DNSResult) string {
	res, _ := sonic.Marshal(d)
	return string(res)
}

func (j jsonOutput) Init() (string, error) {
	return "", nil
}
