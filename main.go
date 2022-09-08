package main

import (
	"github.com/Shib-Chain/shibc-faucet/cmd"
)

//go:generate npm run build
func main() {
	cmd.Execute()
}
