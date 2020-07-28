package main

import (
	"os"

	"github.com/boar-network/billings/cmd"
	"github.com/ipfs/go-log"
	"github.com/urfave/cli"
)

var logger = log.Logger("billings-main")

func main() {
	_ = log.SetLogLevel("*", "DEBUG")

	app := cli.NewApp()

	app.Usage = "KEEP staker billing report generator"

	app.Commands = []cli.Command{
		cmd.BillingsCommand,
	}

	err := app.Run(os.Args)
	if err != nil {
		logger.Fatal(err)
	}
}
