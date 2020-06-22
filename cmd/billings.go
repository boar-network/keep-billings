package cmd

import (
	"encoding/json"
	"fmt"
	"github.com/boar-network/reports/pkg/billing"
	"github.com/boar-network/reports/pkg/exporter"
	"github.com/ipfs/go-log"
	"github.com/urfave/cli"
	"io/ioutil"
	"os"
	"strings"
)

var logger = log.Logger("reports-cmd")

const (
	defaultCustomersJson = "./configs/customers.json"
	defaultTargetDir     = "./generated-billings"
	defaultTemplate      = "./templates/billing_template.html"
)

var BillingsCommand = cli.Command{
	Name:   "billings",
	Action: GenerateBillings,
	Usage:  "Generates billing reports for provided customers",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "customers-json,cj",
			Value: defaultCustomersJson,
			Usage: "JSON file containing customers data",
		},
		&cli.StringFlag{
			Name:  "target-dir,td",
			Value: defaultTargetDir,
			Usage: "Target directory for generated reports",
		},
		&cli.StringFlag{
			Name:  "template,t",
			Value: defaultTemplate,
			Usage: "Template used for generated reports",
		},
	},
}

func GenerateBillings(c *cli.Context) error {
	customersJson := c.String("customers-json")
	targetDir := c.String("target-dir")
	template := c.String("template")

	logger.Infof(
		"generating billings for customers [%v] "+
			"with target dir [%v] "+
			"using template [%v]",
		customersJson,
		targetDir,
		template,
	)

	customersJsonBytes, err := ioutil.ReadFile(customersJson)
	if err != nil {
		return err
	}

	var customers []billing.Customer

	if err := json.Unmarshal(customersJsonBytes, &customers); err != nil {
		return err
	}

	if _, err := os.Stat(targetDir); os.IsNotExist(err) {
		_ = os.Mkdir(targetDir, 0777)
	}

	pdfExporter, err := exporter.NewPdfExporter(template)
	if err != nil {
		return err
	}

	// TODO: trigger each iteration asynchronously.
	for _, customer := range customers {
		logger.Infof("generating billing for [%v]", customer.Name)

		report, err := billing.GenerateReport(&customer)
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
			"%v_Billing.pdf",
			strings.ReplaceAll(customer.Name, " ", "_"),
		)

		err = ioutil.WriteFile(targetDir+"/"+fileName, fileBytes, 0666)
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
