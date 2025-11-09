package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
)

var (
	ErrParseCIDR = errors.New("bad trusted subnet")
)

// TrustedSubnet own type net.IPNet
type TrustedSubnet struct {
	Data *net.IPNet `env:"-" json:"-"`
}

// String flag.Value interface for type TrustedSubnet
func (tsn *TrustedSubnet) String() string {
	n := (*net.IPNet)(tsn.Data)
	return n.String()
}

// Set flag.Value interface for type TrustedSubnet
func (tsn *TrustedSubnet) Set(val string) error {
	_, net, err := net.ParseCIDR(val)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrParseCIDR, err)
	}
	*tsn = TrustedSubnet{Data: net}
	return nil
}

// UnmarshalJSON for TrustedSubnet
func (tsn *TrustedSubnet) UnmarshalJSON(data []byte) error {
	var s string
	err := json.Unmarshal(data, &s)
	if err != nil {
		return err
	}

	return tsn.Set(s)
}

// IsTrusted check ip in trusted net
func (tsn *TrustedSubnet) IsTrusted(ip string) bool {
	if tsn.Data == nil {
		return false
	}
	pip := net.ParseIP(ip)
	return pip != nil && tsn.Data.Contains(pip)
}
