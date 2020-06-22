package cmd

import (
	"encoding/json"
	"fmt"
	"github.com/boar-network/reports/pkg/billing"
	"github.com/boar-network/reports/pkg/chain"
	"github.com/boar-network/reports/pkg/exporter"
	"github.com/ipfs/go-log"
	"github.com/urfave/cli"
	"io/ioutil"
	"os"
	"strings"
)

var logger = log.Logger("reports-cmd")

const (
	defaultConfigFile = "./configs/config.toml"
)

var BillingsCommand = cli.Command{
	Name:   "billings",
	Action: GenerateBillings,
	Usage:  "Generates billing reports for provided customers",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "config,c",
			Value: defaultConfigFile,
			Usage: "Path to the TOML config file",
		},
	},
}

func GenerateBillings(c *cli.Context) error {
	configPath := c.String("config")

	logger.Infof("generating billings using config [%v]", configPath)

	config, err := ReadConfig(configPath)
	if err != nil {
		return err
	}

	customersJsonBytes, err := ioutil.ReadFile(config.Billings.CustomersFile)
	if err != nil {
		return err
	}

	var customers []billing.Customer

	if err := json.Unmarshal(customersJsonBytes, &customers); err != nil {
		return err
	}

	if _, err := os.Stat(config.Billings.TargetDirectory); os.IsNotExist(err) {
		_ = os.Mkdir(config.Billings.TargetDirectory, 0777)
	}

	pdfExporter, err := exporter.NewPdfExporter(config.Billings.TemplateFile)
	if err != nil {
		return err
	}

	ethereumClient, err := chain.NewEthereumClient(
		config.Ethereum.URL,
		config.Ethereum.KeepRandomBeaconOperator,
	)
	if err != nil {
		return err
	}

	// TODO: trigger each iteration asynchronously.
	for _, customer := range customers {
		logger.Infof("generating billing for [%v]", customer.Name)

		report, err := billing.GenerateReport(&customer, ethereumClient)
		if err != nil {
			logger.Errorf(
				"could not generate billing report for customer [%v]: [%v]",
				customer.Name,
				err,
			)
			continue
		}

		fileBytes, err := pdfExporter.Export(report)
		if err != nil {
			logger.Errorf(
				"could not export billing pdf for customer [%v]: [%v]",
				customer.Name,
				err,
			)
			continue
		}

		fileName := fmt.Sprintf(
			"%v/%v_Billing.pdf",
			config.Billings.TargetDirectory,
			strings.ReplaceAll(customer.Name, " ", "_"),
		)

		err = ioutil.WriteFile(fileName, fileBytes, 0666)
		if err != nil {
			logger.Errorf(
				"could not write billing pdf file for customer [%v]: [%v]",
				customer.Name,
				err,
			)
			continue
		}

		logger.Infof("completed billing for [%v]", customer.Name)
	}

	return nil
}
