

# CasperAPI provider application.
CasperAPI is a decentralized data storage.
We were deeply concerned with the state of the Internet and thought that with the state-of-art decentralized technologies we could at least start changing the way data is stored and distributed right now.

## Table of Contents

- [Building from source](#building-from-source)
  - [Debian-based](#debian-based-linux)
- [Usage](#usage)
- [Special thanks](#special-thanks)
- [License](#license)
  
## Building from source
### Debian-based linux
#### Prerequisites
For the build process you'll need Go 1.9.2 or higher. We also assume that you already exported $GOROOT and $GOPATH variables and have $GOROOT/bin exported to your $PATH environment variable.

```bash
apt-get install build-essential \
	software-properties-common \
	g++ \
	gcc \
	cmake \
	libboost-all-dev \
	git \
	libz3-dev
apt-get update
```

#### Installation
Get dependencies and Casper-server via go get
```bash
go get -v -u -d github.com/Casper-dev/Casper-server
go get -v -u -d github.com/Casper-dev/Casper-SC
go get -v -u -d github.com/whyrusleeping/gx
go get -v -u -d github.com/whyrusleeping/gx-go 
go get -v -u -d github.com/ethereum/go-ethereum
go get -v git.apache.org/thrift.git/lib/go/thrift/...
```
Install [Solidity](https://solidity.readthedocs.io/en/latest/installing-solidity.html#) compiler

Install gx from sources
```bash
cd $GOPATH/src/github.com/whyrusleeping/gx
go install
cd $GOPATH/src/github.com/whyrusleeping/gx-go
go install
```
Install ipfs dependencies via gx
```bash
cd $GOPATH/src/github.com/Casper-dev/Casper-server
gx --verbose install --global
chmod +x bin/*
```
Install casper-server
```bash
cd $GOPATH/src/github.com/Casper-dev/Casper-server/cmd/ipfs
go install
```
Now you got casper-server named ipfs at $GOPATH/bin
```bash
cd $GOPATH/bin
file ipfs
ipfs: ELF 64-bit LSB executable, x86-64, version 1 (SYSV), dynamically linked, interpreter /lib64/ld-linux-x86-64.so.2, for GNU/Linux 2.6.32, BuildID[sha1]=2478eaaff91f2846ccfcef826de7d74f4261ed13, not stripped
```

#### Usage
Provider mostly just serves incoming connections from client, so you only need to run it. As of now, works only with open IP addresses and ~~requires own INFURA Ropsten test network ethereum wallet and some ETH on it~~ for now works on fast private blockchain.
```bash
ipfs init 	 			# this will instantiate an id and a repo that provider will use
ipfs daemon	 			# runs ipfs daemon that will serve incoming commands
# or you can use
ipfs daemon --init=true # runs ipfs daemon even without previous ipfs init; will make an id and a repo if there's none already instanced
```

# Special thanks
We really appreciate all the work that IPFS team done to the moment. 
Guys - you rock!

# License
[Proprietary](LICENSE)