package cmd

import (
	"fmt"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Billings Billings
	Ethereum Ethereum
}

type Billings struct {
	CustomersFile      string
	TargetDirectory    string
	BeaconTemplateFile string
	EcdsaTemplateFile  string
}

type Ethereum struct {
	URL                      string
	KeepToken                string
	KeepRandomBeaconOperator string
	BondedECDSAKeepFactory   string
}

func ReadConfig(filePath string) (*Config, error) {
	config := &Config{}

	if _, err := toml.DecodeFile(filePath, config); err != nil {
		return nil, fmt.Errorf(
			"could not decode config file [%v]: [%v]",
			filePath,
			err,
		)
	}

	return config, nil
}
