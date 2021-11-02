package pool

type StorageConfig struct {
	Enabled           bool   `json:"enabled"`
	Endpoint          string `json:"endpoint"`
	PasswordEncrypted string `json:"passwordEncrypted"`
	Password          string `json:"-"`
	Database          int64  `json:"database"`
	PoolSize          int    `json:"poolSize"`
}

type StorageConfigFailover struct {
	Enabled           bool     `json:"enabled"`
	MasterName        string   `json:"masterName"`
	SentinelEndpoints []string `json:"sentinelEndpoints"`
	PasswordEncrypted string   `json:"passwordEncrypted"`
	Password          string   `json:"-"`
	Database          int64    `json:"database"`
	PoolSize          int      `json:"poolSize"`
}

type UnlockerConfig struct {
	Enabled        bool    `json:"enabled"`
	PoolFee        float64 `json:"poolFee"`
	PoolFeeAddress string  `json:"poolFeeAddress"`
	Donate         bool    `json:"donate"`
	Depth          int64   `json:"depth"`
	ImmatureDepth  int64   `json:"immatureDepth"`
	KeepTxFees     bool    `json:"keepTxFees"`
	Interval       string  `json:"interval"`
	DaemonName     string  `json:"daemonName"`
	DaemonHost     string  `json:"daemonHost"`
	DaemonPort     int     `json:"daemonPort"`
	Timeout        string  `json:"timeout"`
}

type Config struct {
	AddressEncrypted        string     `json:"addressEncrypted"`
	Address                 string     `json:"-"`
	BypassAddressValidation bool       `json:"bypassAddressValidation"`
	BypassShareValidation   bool       `json:"bypassShareValidation"`
	Log                     Log        `json:"log"`
	Stratum                 Stratum    `json:"stratum"`
	StratumTls              StratumTls `json:"stratumTls"`
	BlockRefreshInterval    string     `json:"blockRefreshInterval"`
	UpstreamCheckInterval   string     `json:"upstreamCheckInterval"`
	Upstream                []Upstream `json:"upstream"`
	EstimationWindow        string     `json:"estimationWindow"`
	LuckWindow              string     `json:"luckWindow"`
	LargeLuckWindow         string     `json:"largeLuckWindow"`
	HashRateExpiration      string     `json:"hashRateExpiration"`

	PurgeInterval       string `json:"purgeInterval"`
	HashrateWindow      string `json:"hashrateWindow"`
	HashrateLargeWindow string `json:"hashrateLargeWindow"`

	Threads  int      `json:"threads"`
	Frontend Frontend `json:"frontend"`

	Coin          string                `json:"coin"`
	Redis         StorageConfig         `json:"redis"`
	RedisFailover StorageConfigFailover `json:"redisFailover"`

	BlockUnlocker UnlockerConfig `json:"unlocker"`

	NewrelicName    string `json:"newrelicName"`
	NewrelicKey     string `json:"newrelicKey"`
	NewrelicVerbose bool   `json:"newrelicVerbose"`
	NewrelicEnabled bool   `json:"newrelicEnabled"`
}

type Stratum struct {
	Enabled bool   `json:"enabled"`
	Timeout string `json:"timeout"`
	Ports   []Port `json:"listen"`
}

type Port struct {
	Difficulty int64  `json:"diff"`
	Host       string `json:"host"`
	Port       int    `json:"port"`
	MaxConn    int    `json:"maxConn"`
}

type StratumTls struct {
	Enabled bool   `json:"enabled"`
	Timeout string `json:"timeout"`
	Ports   []Port `json:"listen"`
	TlsCert string `json:"tlsCert"`
	TlsKey  string `json:"tlsKey"`
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
