# xmrpool

High performance CryptoNote mining stratum with Web-interface written in Golang.


**Stratum feature list:**

* Be your own pool
* Rigs availability monitoring
* Keep track of accepts, rejects, blocks stats
* Easy detection of sick rigs
* Daemon failover list
* Concurrent shares processing
* Beautiful Web-interface


## Installation

Dependencies:

  * go-1.17
  * Everything required to build Monero
  * Monero >= **v0.18.3.1** (sometimes `master` branch required)

### Linux

Use Ubuntu 16.04 LTS.

    sudo apt-get install libssl-dev
    sudo apt-get install git cmake build-essential pkg-config libboost-all-dev libreadline-dev doxygen libsodium-dev libzmq5-dev
    sudo apt-get install liblmdb-dev libevent-dev libjson-c-dev uuid-dev libunbound-dev

Use Ubuntu 18.04 LTS.

    sudo apt-get install libssl1.0-dev
    sudo apt-get install git cmake build-essential pkg-config libboost-all-dev libreadline-dev doxygen libsodium-dev libzmq5-dev 
    sudo apt-get install liblmdb-dev libevent-dev libjson-c-dev uuid-dev libunbound-dev


Compile Monero source (with shared libraries option):

    git clone --recursive https://github.com/monero-project/monero.git
    cd monero
    git checkout tags/v0.18.3.1 -b v0.18.3.1
    cmake -DBUILD_SHARED_LIBS=1 -DMANUAL_SUBMODULES=1 .
    make

Install Golang and required packages:

    sudo apt install software-properties-common
    sudo add-apt-repository ppa:longsleep/golang-backports
    sudo apt-get update
    sudo apt-get install golang-go

Clone:

    git clone https://github.com/PowPool/xmrpool.git
    cd xmrpool

Build stratum:

    export MONERO_DIR=[path_of_monero] 
    cmake .
    make
    make -f Makefile_build_info

`MONERO_DIR=/path/to/monero` is optional, not needed if both `monero` and `xmrpool` is in the same directory like `/opt/src/`. By default make will search for monero libraries in `../monero`. You can just run `cmake .`.

### Mac OS X

Compile Monero source:

    git clone --recursive https://github.com/monero-project/monero.git
    cd monero
    git checkout tags/v0.18.3.1 -b v0.18.3.1
    cmake .
    make

Install Golang and required packages:

    brew update && brew install go

Clone stratum:

    git clone https://github.com/PowPool/xmrpool.git
    cd xmrpool

Build stratum:

    MONERO_DIR=[path_of_monero]  
    go mod tidy -compat="1.17"
    cmake .
    make
    make -f Makefile_build_info

### Running Stratum

    ./build/bin/xmrpool config.json

* About `Security Password`:

We use `Security Password` to prevent our important configuration information from being viewed or modified.
So we encrypt some important configurations in config.json.

    addressEncrypted
    redis.passwordEncrypted or redisFailover.passwordEncrypted

* How to encrypt/decrypt

Use function `TestAe64Encode` and `TestAe64Decode` from model `util` in this project.


If you need to bind to privileged ports and don't want to run from `root`:

    sudo apt-get install libcap2-bin
    sudo setcap 'cap_net_bind_service=+ep' /path/to/xmrpool

## Configuration

Configuration is self-describing, just copy *config.example.json* to *config.json* and run stratum with path to config file as 1st argument.

```json
{
	// Address for block rewards
	"addressEncrypted": "xePkFW8E5OhMuFg/FyQQqCFs2awuArs/QGEAyJcZ8X4mBV6FM+k1vER2WW6lMSUd1cP/ggTXR8j+43qVa+bNijZMdZyrohoT/7rNUj/toDFGVxzDTDtrRvVs9LbRGtIx",
	// Don't validate address
	"bypassAddressValidation": false,
	// Don't validate shares
	"bypassShareValidation": false,

	"threads": 16,
	"coin": "xmr",

	"estimationWindow": "15m",
	"luckWindow": "24h",
	"largeLuckWindow": "72h",

	// Interval to poll daemon for new jobs
	"blockRefreshInterval": "1s",
	"hashRateExpiration": "3h",

	"purgeInterval": "10m",
	"hashrateWindow": "30m",
	"hashrateLargeWindow": "3h",

	"log": {
		"logSetLevel": 10
	},

	"stratum": {
		"enabled": true,
		// Socket timeout
		"timeout": "15m",
		"listen": [
			{
				"host": "0.0.0.0",
				"port": 3003,
				"diff": 300000,
				"maxConn": 50000
			}
		]
	},

	"stratumTls": {
		"enabled": false,
		// Socket timeout
		"timeout": "15m",
		"listen": [
			{
				"host": "0.0.0.0",
				"port": 13003,
				"diff": 300000,
				"maxConn": 50000
			}
		],

		"tlsCert": "certs/server.pem",
		"tlsKey": "certs/server.key"
	},

	"frontend": {
		"enabled": false,
		"listen": "0.0.0.0:8082",
		"login": "admin",
		"password": "",
		"hideIP": false
	},

	"upstreamCheckInterval": "5s",
	// upstream to monerod rpc-api, multiple monerod supported
	"upstream": [
		{
			"name": "Main",
			"host": "192.168.33.166",
			"port": 18081,
			"timeout": "3s"
		},
		{
			"name": "Backup1",
			"host": "192.168.26.83",
			"port": 18081,
			"timeout": "3s"
		},
		{
			"name": "Backup2",
			"host": "192.168.33.243",
			"port": 18081,
			"timeout": "3s"
		}
	],
	// redis single node mode
	"redis": {
		"enabled": true,
		"endpoint": "127.0.0.1:6379",
		"poolSize": 10,
		"database": 0,
		"passwordEncrypted": "oXyI5OTy+nRTshESi80X8KKSjDiLksuw1mhwRg2z0Ic="
	},
	// redis failover mode
	"redisFailover": {
		"enabled": false,
		"masterName": "mymaster",
		"sentinelEndpoints": ["192.168.33.166:26379", "192.168.26.83:26379", "192.168.33.243:26379"],
		"poolSize": 10,
		"database": 0,
		"passwordEncrypted": "oXyI5OTy+nRTshESi80X8KKSjDiLksuw1mhwRg2z0Ic="
	},

	"unlocker": {
		"enabled": true,
		"poolFee": 1.0,
		"poolFeeAddress": "",
		"donate": false,
		"depth": 6,
		"immatureDepth": 3,
		"keepTxFees": false,
		"interval": "10m",
		"daemonName": "unlocker",
		"daemonHost": "192.168.33.166",
		"daemonPort": 18081,
		"timeout": "10s"
	},

	"newrelicEnabled": false,
	"newrelicName": "MyStratum",
	"newrelicKey": "SECRET_KEY",
	"newrelicVerbose": false
}
```

You must use `anything.WorkerID` as username in your miner. Either disable address validation or use `<address>.WorkerID` as username. If there is no workerID specified your rig stats will be merged under `0` worker. If mining software contains dev fee rounds its stats will usually appear under `0` worker. This stratum acts like your own pool, the only exception is that you will get rewarded only after block found, shares only used for stats.


### Mining tools

* [xmrig](https://github.com/xmrig/xmrig)
* [xmr-stak](https://github.com/fireice-uk/xmr-stak)


### License

Released under the GNU General Public License v2.

http://www.gnu.org/licenses/gpl-2.0.html
