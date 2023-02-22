package config

import (
	"encoding/json"
	"fmt"
	"gdut-drcom-go/lib/types"
	"net/netip"
)

type Config struct {
	LogFile string                     `json:"log_file"`
	Debug   bool                       `json:"debug"`
	Core    types.Listable[ConfigCore] `json:"core"`
}

type ConfigCore struct {
	Tag            string     `json:"tag"`
	RemoteIP       netip.Addr `json:"remote_ip"`
	RemotePort     uint16     `json:"remote_port"`
	KeepAlive1Flag byte       `json:"keep_alive1_flag"`
	EnableCrypt    bool       `json:"enable_crypt"`
	BindDevice     string     `json:"bind_device"`
	BindToAddr     bool       `json:"bind_to_addr"`
}

func (c *ConfigCore) UnmarshalJSON(data []byte) error {
	var _c ConfigCore
	err := json.Unmarshal(data, &_c)
	if err != nil {
		return err
	}
	if !_c.RemoteIP.Is4() && !_c.RemoteIP.Is6() {
		return fmt.Errorf("invalid remote ip: %s", _c.RemoteIP)
	}
	if _c.RemotePort == 0 {
		_c.RemotePort = 61440
	}
	if _c.KeepAlive1Flag == 0 {
		_c.KeepAlive1Flag = 0xdc
	}
	*c = _c
	return nil
}
