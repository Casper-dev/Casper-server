# CasperAPI provider application.
CasperAPI is a decentralized data storage.
We were deeply concerned with the state of the Internet and thought that with the state-of-art decentralized technologies we could at least start changing the way data is stored and distributed right now.

## Table of Contents

- [Install](#building-from-source)
  - [Building from source on Windows](#windows-platforms)
  - [Building from source on Linux](#Linux-platforms)
    - [Debian-based](#Debian-based)
- [Usage](#usage)
- [Special thanks](#special-thanks)
- [License](#license)
  
## Building from source
### Windows-platforms
For the build process you'll need Go 1.9 or higher and mingw cpp compiler 4.5 or higher.
Later steps assume that you set GOPATH and GOROOT environment variables, and added %GOPATH%/bin and %GOROOT%/bin to PATH
Download dependencies
```
go get -u -d github.com/Casper-dev/Casper-SC
go get -u -d github.com/whyrusleeping/gx
go get -u -d github.com/whyrusleeping/gx-go
go get -u -d github.com/ethereum/go-ethereum
go get -v git.apache.org/thrift.git/lib/go/thrift/...
```
Download solc binaries from https://github.com/ethereum/solidity/releases unpack and add folder containing solc.exe to PATH

Download all ipfs deps using gx
```
cd %GOPATH%/src/github.com/ipfs/go-ipfs
gx --verbose install --global
```

If gx wasn't installed, run 
```
go install %GOPATH%/src/github/whyrusleeping/gx
go install %GOPATH%/src/github/whyrusleeping/gx-go
```

To have access to our SC you'll also need a wallet with some ETH on INFURA ropsten network.
Paste your private key and INFURA access keys into github.com/Casper-dev/Casper-SC/casper_sc/casper_sc.go

Build ipfs using go install
```
cd %GOPATH%/src/github.com/ipfs/go-ipfs/cmd/ipfs
go install
```


And now you got your own CasperAPI provider app built from sources.
### Linux platforms
#### Prerequisites
* Golang 1.9.* and higher. We're using Golang 1.9.4. [Download](https://golang.org/dl/).
* Check your Go environment. GOPATH, GOROOT, PATH should be set up.

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

## Usage
Run ipfs daemon. 
Server app mostly spews logs to >1 and waits for client connection, but it's somewhat can 

# Special thanks
We really appreciate all the work that IPFS team done to the moment. 
Guys - you rock!

# License
[Proprietary](LICENSE)