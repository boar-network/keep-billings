package billing

import (
	"math/big"

	"github.com/ipfs/go-log"
)

var logger = log.Logger("billings-billing")

type Customer struct {
	Name                      string
	Operator                  string
	Beneficiary               string
	CustomerSharePercentage   int
	InitialOperatorEthBalance int
}

type Report struct {
	Customer *Customer

	Stake                  string
	OperatorBalance        string
	BeneficiaryEthBalance  string
	BeneficiaryKeepBalance string
	BeneficiaryTbtcBalance string

	AccumulatedRewards string
	OperationalCosts   string
	CustomerEthEarned  string
	ProviderEthEarned  string
}

type DataSource interface {
	EthBalance(address string) (*big.Float, error)
	Stake(address string) (*big.Float, error)
	KeepBalance(address string) (*big.Float, error)
	TbtcBalance(address string) (*big.Float, error)
}
