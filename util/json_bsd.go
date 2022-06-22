//go:build openbsd || freebsd

package util

import (
	"github.com/bytedance/sonic"
)

type JsonOutput struct{}

func (j JsonOutput) Marshal(d DNSResult) string {
	res, _ := sonic.Marshal(d)
	return string(res)
}

func (j JsonOutput) Init() (string, error) {
	return "", nil
}
