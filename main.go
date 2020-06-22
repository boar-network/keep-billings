package main

import (
	"github.com/boar-network/reports/cmd"
	"github.com/ipfs/go-log"
	"github.com/urfave/cli"
	"os"
)

var logger = log.Logger("reports-main")

func main() {
	_ = log.SetLogLevel("*", "DEBUG")

	app := cli.NewApp()

	app.Usage = "reporting tools"

	app.Commands = []cli.Command{
		cmd.BillingsCommand,
	}

	err := app.Run(os.Args)
	if err != nil {
		logger.Fatal(err)
	}
}
