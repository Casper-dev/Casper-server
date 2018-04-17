## ATTENTION 
As of now we are not providing you folder "vendor", so server won't build, and nor client. Sorry for the inconvenience, we'll think of something the following week.
# CasperAPI provider application.
CasperAPI is a decentralized data storage.
We were deeply concerned with the state of the Internet and thought that with the state-of-art decentralized technologies we could at least start changing the way data is stored and distributed right now.

## Table of Contents
- [Installation](#installation)
  - [Building from source](#building-from-source)
    - [Debian-based](#debian-based-linux)
- [Getting started](#getting-started)
- [Special thanks](#special-thanks)
- [License](#license)
  
## Installation
You can download and use pre-built binaries (download them [here](https://github.com/Casper-dev/Casper-server/releases/tag/0.0.1)). If there's none for your OS or you want to build everything from scratch please use the instruction below.
For instructions on how to run your own node see [Getting started](#getting-started).

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

#### Getting started
Provider mostly just serves incoming connections from client, so you only need to run it. 
If you are using a pre-built ipfs binary, you need to unpack it (default on Windows; on Linux use tar xvf <filename>), place contents somewhere in the system and then open a terminal(or PowerShell) near and run
```bash
./ipfs init 	 			# this will instantiate an id and a repo that provider will use
./ipfs daemon	 			# runs ipfs daemon that will serve incoming commands
# or you can use
./ipfs daemon --init=true # runs ipfs daemon even without previous ipfs init; will make an id and a repo if there's none already instanced
```
If you built binaries yourself then look for them in $GOPATH/bin.

#### Current issues
As of now, works only with open IP addresses and ~~requires own INFURA Ropsten test network ethereum wallet and some ETH on it~~ for now works on fast private blockchain.

# Special thanks
We really appreciate all the work that IPFS team done to the moment. 
Guys - you rock!

# License
[Proprietary](LICENSE)