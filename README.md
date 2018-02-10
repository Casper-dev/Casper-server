# CasperAPI provider application.
CasperAPI is a decentralized data storage.
We were deeply concerned with the state of the Internet and thought that with the state-of-art decentralized technologies we could at least start changing the way data is stored and distributed right now.

## Table of Contents

- [Install](#building-from-source)
  - [Building from source on windows](#windows-platforms)
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
TODO

## Usage
Run ipfs daemon. 
Server app mostly spews logs to >1 and waits for client connection, but it's somewhat can 

# Special thanks
We really appreciate all the work that IPFS team done to the moment. 
Guys - you rock!

# License
[Proprietary](LICENSE)