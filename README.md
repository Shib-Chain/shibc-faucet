# shibc-faucet

[![Go](https://img.shields.io/github/go-mod/go-version/Shib-Chain/shibc-faucet)](https://go.dev/)

The faucet is a web application with the goal of distributing small amounts of Shibc in private and test networks.

## Features

* Allow to configure the funding account via private key or keystore
* Asynchronous processing Txs to achieve parallel execution of user requests
* Rate limiting by SHIBC address and IP address as a precaution against spam
* Prevent X-Forwarded-For spoofing by specifying the count of reverse proxies

## Get started

### Prerequisites

* Go (1.18 or later)
* Node.js

### Installation

1. Clone the repository and navigate to the app’s directory
```bash
git clone https://github.com/Shib-Chain/shibc-faucet.git
cd shibc-faucet
```

2. Bundle Frontend web with Vite
```bash
npm run build
```

3. Build Go project 
```bash
go build -o shibc-faucet
```

## Usage

**Use private key to fund users**

```bash
./shibc-faucet -httpport 8080 -wallet.provider http://localhost:8545 -wallet.privkey privkey
```

**Use keystore to fund users**

```bash
./shibc-faucet -httpport 8080 -wallet.provider http://localhost:8545 -wallet.keyjson keystore -wallet.keypass password.txt
```

### Configuration

You can configure the funder by using environment variables instead of command-line flags as follows:
```bash
export WEB3_PROVIDER=rpc endpoint
export PRIVATE_KEY=hex private key
```

or

```bash
export WEB3_PROVIDER=rpc endpoint
export KEYSTORE=keystore path
echo "your keystore password" > `pwd`/password.txt
```

Then run the faucet application without the wallet command-line flags:
```bash
./shibc-faucet -httpport 8080
```

**Optional Flags**

The following are the available command-line flags(excluding above wallet flags):

| Flag           | Description                                      | Default Value
| -------------- | ------------------------------------------------ | -------------
| -httpport      | Listener port to serve HTTP connection           | 8080
| -proxycount    | Count of reverse proxies in front of the server  | 1
| -queuecap      | Maximum transactions waiting to be sent          | 100
| -faucet.amount | Number of WSHIBs to transfer per user request    | 1
| -faucet.minutes| Number of minutes to wait between funding rounds | 1440
| -faucet.name   | Network name to display on the frontend          | shibc-testnet

### Docker deployment

```bash
docker run -d -p 8080:8080 -e WEB3_PROVIDER=rpc endpoint -e PRIVATE_KEY=hex private key Shib-Chain/shibc-faucet:1.1.0
```

or

```bash
docker run -d -p 8080:8080 -e WEB3_PROVIDER=rpc endpoint -e KEYSTORE=keystore path -v `pwd`/keystore:/app/keystore -v `pwd`/password.txt:/app/password.txt Shib-Chain/shibc-faucet:1.1.0
```
