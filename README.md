# CasperAPI provider application.
CasperAPI is a decentralized data storage.
We were deeply concerned with the state of the Internet and thought that with the state-of-art decentralized technologies we could at least start changing the way data is stored and distributed right now.

## Table of Contents

- [Install](#building-from-source)
  - [Building from source on Linux](#Linux-platforms)
    - [Debian-based](#Debian-based)
- [Usage](#usage)
- [Special thanks](#special-thanks)
- [License](#license)
  
## Building from source
### Building from source on Linux
#### Debian-based
##### Prerequisites
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

##### Installation
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

##### Usage
```bash
ipfs init
ipfs daemon
# OR
ipfs daemon --init=true #Initialize ipfs with default settings if not already initialized.
```