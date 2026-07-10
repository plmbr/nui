// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package devcontainer

import (
	"net"
	"net/url"
)

func loopbackHostArgs(baseURL string) []string {
	if baseURL == "" {
		return nil
	}
	u, err := url.Parse(baseURL)
	if err != nil || u.Hostname() == "" {
		return nil
	}
	hostname := u.Hostname()
	addrs, err := net.LookupHost(hostname)
	if err != nil {
		return nil
	}
	for _, addr := range addrs {
		if ip := net.ParseIP(addr); ip != nil && ip.IsLoopback() {
			return []string{"--add-host", hostname + ":host-gateway"}
		}
	}
	return nil
}
