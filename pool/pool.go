package pool

import "github.com/MiningPool0826/xmrpool/storage"

type Config struct {
	AddressEncrypted        string     `json:"addressEncrypted"`
	Address                 string     `json:"-"`
	BypassAddressValidation bool       `json:"bypassAddressValidation"`
	BypassShareValidation   bool       `json:"bypassShareValidation"`
	Log                     Log        `json:"log"`
	Stratum                 Stratum    `json:"stratum"`
	BlockRefreshInterval    string     `json:"blockRefreshInterval"`
	UpstreamCheckInterval   string     `json:"upstreamCheckInterval"`
	Upstream                []Upstream `json:"upstream"`
	EstimationWindow        string     `json:"estimationWindow"`
	LuckWindow              string     `json:"luckWindow"`
	LargeLuckWindow         string     `json:"largeLuckWindow"`
	HashRateExpiration      string     `json:"hashRateExpiration"`
	Threads                 int        `json:"threads"`
	Frontend                Frontend   `json:"frontend"`

	Coin          string                 `json:"coin"`
	Redis         storage.Config         `json:"redis"`
	RedisFailover storage.ConfigFailover `json:"redisFailover"`

	NewrelicName    string `json:"newrelicName"`
	NewrelicKey     string `json:"newrelicKey"`
	NewrelicVerbose bool   `json:"newrelicVerbose"`
	NewrelicEnabled bool   `json:"newrelicEnabled"`
}

type Stratum struct {
	Timeout string `json:"timeout"`
	Ports   []Port `json:"listen"`
}

type Port struct {
	Difficulty int64  `json:"diff"`
	Host       string `json:"host"`
	Port       int    `json:"port"`
	MaxConn    int    `json:"maxConn"`
}

type Upstream struct {
	Name    string `json:"name"`
	Host    string `json:"host"`
	Port    int    `json:"port"`
	Timeout string `json:"timeout"`
}

type Frontend struct {
	Enabled  bool   `json:"enabled"`
	Listen   string `json:"listen"`
	Login    string `json:"login"`
	Password string `json:"password"`
	HideIP   bool   `json:"hideIP"`
}

type Log struct {
	LogSetLevel int `json:"logSetLevel"`
}
