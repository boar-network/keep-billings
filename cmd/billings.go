package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/boar-network/keep-billings/pkg/billing"
	"github.com/boar-network/keep-billings/pkg/chain"
	"github.com/boar-network/keep-billings/pkg/exporter"
	"github.com/ipfs/go-log"
	"github.com/urfave/cli"
)

var logger = log.Logger("billings-cmd")

const (
	defaultConfigFile = "./configs/config.toml"
)

var BillingsCommand = cli.Command{
	Name:   "generate",
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

type Customers struct {
	Beacon []billing.Customer
	Ecdsa  []billing.Customer
}

func GenerateBillings(c *cli.Context) error {
	configPath := c.String("config")

	logger.Infof("generating billings using config [%v]", configPath)

	config, err := ReadConfig(configPath)
	if err != nil {
		return err
	}

	customers, err := parseCustomers(config)
	if err != nil {
		return err
	}

	createTargetDirectory(config)

	ethereumClient, err := chain.NewEthereumClient(
		config.Ethereum.URL,
		config.Ethereum.KeepToken,
		config.Ethereum.TokenStaking,
		config.Ethereum.KeepRandomBeaconOperator,
	)
	if err != nil {
		return err
	}

	beaconReportGenerator := billing.NewBeaconReportGenerator(ethereumClient)

	beaconPdfExporter, err := exporter.NewPdfExporter(
		config.Billings.BeaconTemplateFile,
	)
	if err != nil {
		return err
	}

	generateBillings(
		customers.Beacon,
		beaconReportGenerator.FetchCommonData,
		func(customer *billing.Customer) (interface{}, error) {
			return beaconReportGenerator.Generate(customer)
		},
		beaconPdfExporter,
		config.Billings.TargetDirectory+"/%v_Beacon_Billing.pdf",
	)

	return nil
}

func parseCustomers(config *Config) (*Customers, error) {
	customersJsonBytes, err := ioutil.ReadFile(config.Billings.CustomersFile)
	if err != nil {
		return nil, err
	}

	var customers Customers
	if err := json.Unmarshal(customersJsonBytes, &customers); err != nil {
		return nil, err
	}

	return &customers, nil
}

func createTargetDirectory(config *Config) {
	if _, err := os.Stat(config.Billings.TargetDirectory); os.IsNotExist(err) {
		_ = os.Mkdir(config.Billings.TargetDirectory, 0777)
	}
}

func generateBillings(
	customers []billing.Customer,
	setUp func() error,
	generate func(customer *billing.Customer) (interface{}, error),
	pdfExporter *exporter.PdfExporter,
	fileNameFormat string,
) {
	if len(customers) == 0 {
		logger.Infof("no customers to generate the report for, quitting")
		return
	}

	if err := setUp(); err != nil {
		logger.Errorf("could not set up generator: [%v]", err)
		return
	}

	for _, customer := range customers {
		logger.Infof("generating billing for [%v]", customer.Name)

		report, err := generate(&customer)
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
			fileNameFormat,
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
}
