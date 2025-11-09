package config

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsTrusted(t *testing.T) {
	tests := []struct {
		name          string
		trustedSubNet TrustedSubnet
		ip            string
		isTrusted     bool
	}{
		{
			name:          "trusted net not defined. ip empty",
			trustedSubNet: TrustedSubnet{},
			ip:            "",
			isTrusted:     false,
		},
		{
			name:          "trusted net not defined. ip not valid",
			trustedSubNet: TrustedSubnet{},
			ip:            "not valid",
			isTrusted:     false,
		},
		{
			name:          "trusted net not defined. ip valid",
			trustedSubNet: TrustedSubnet{},
			ip:            "192.168.0.1",
			isTrusted:     false,
		},

		{
			name: "trusted net defined. ip empty",
			trustedSubNet: TrustedSubnet{
				Data: &net.IPNet{
					IP:   net.IP([]byte{0x7f, 0x0, 0x0, 0x0}),
					Mask: net.IPMask([]byte{0xff, 0xff, 0xff, 0x0}),
				},
			},
			ip:        "",
			isTrusted: false,
		},
		{
			name: "trusted net defined. ip not valid",
			trustedSubNet: TrustedSubnet{
				Data: &net.IPNet{
					IP:   net.IP([]byte{0x7f, 0x0, 0x0, 0x0}),
					Mask: net.IPMask([]byte{0xff, 0xff, 0xff, 0x0}),
				},
			},
			ip:        "not valid",
			isTrusted: false,
		},
		{
			name: "trusted net defined. ip valid not trusted",
			trustedSubNet: TrustedSubnet{
				Data: &net.IPNet{
					IP:   net.IP([]byte{0x7f, 0x0, 0x0, 0x0}),
					Mask: net.IPMask([]byte{0xff, 0xff, 0xff, 0x0}),
				},
			},
			ip:        "192.168.0.1",
			isTrusted: false,
		},
		{
			name: "trusted net defined. ip valid and trusted",
			trustedSubNet: TrustedSubnet{
				Data: &net.IPNet{
					IP:   net.IP([]byte{0x7f, 0x0, 0x0, 0x0}),
					Mask: net.IPMask([]byte{0xff, 0xff, 0xff, 0x0}),
				},
			},
			ip:        "127.0.0.0",
			isTrusted: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			b := test.trustedSubNet.IsTrusted(test.ip)
			assert.Equal(t, test.isTrusted, b)
		})
	}
}
