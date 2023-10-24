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

If you need to bind to privileged ports and don't want to run from `root`:

    sudo apt-get install libcap2-bin
    sudo setcap 'cap_net_bind_service=+ep' /path/to/xmrpool

## Configuration

Configuration is self-describing, just copy *config.example.json* to *config.json* and run stratum with path to config file as 1st argument.

```javascript
{
  // Address for block rewards
  "address": "YOUR-ADDRESS-NOT-EXCHANGE",
  // Don't validate address
  "bypassAddressValidation": true,
  // Don't validate shares
  "bypassShareValidation": true,

  "threads": 2,

  "estimationWindow": "15m",
  "luckWindow": "24h",
  "largeLuckWindow": "72h",

  // Interval to poll daemon for new jobs
  "blockRefreshInterval": "1s",

  "stratum": {
    // Socket timeout
    "timeout": "15m",

    "listen": [
      {
        "host": "0.0.0.0",
        "port": 1111,
        "diff": 5000,
        "maxConn": 32768
      },
      {
        "host": "0.0.0.0",
        "port": 3333,
        "diff": 10000,
        "maxConn": 32768
      }
    ]
  },

  "frontend": {
    "enabled": true,
    "listen": "0.0.0.0:8082",
    "login": "admin",
    "password": "",
    "hideIP": false
  },

  "upstreamCheckInterval": "5s",

  "upstream": [
    {
      "name": "Main",
      "host": "127.0.0.1",
      "port": 18081,
      "timeout": "10s"
    }
  ]
}
```

You must use `anything.WorkerID` as username in your miner. Either disable address validation or use `<address>.WorkerID` as username. If there is no workerID specified your rig stats will be merged under `0` worker. If mining software contains dev fee rounds its stats will usually appear under `0` worker. This stratum acts like your own pool, the only exception is that you will get rewarded only after block found, shares only used for stats.


### License

Released under the GNU General Public License v2.

http://www.gnu.org/licenses/gpl-2.0.html
