package billing

import "math/big"

type Customer struct {
	Name                    string
	Operator                string
	Beneficiary             string
	CustomerSharePercentage int
}

type Report struct {
	Customer *Customer

	OperatorBalance    string
	BeneficiaryBalance string

	AccumulatedRewards string
}

type DataSource interface {
	GetEthBalance(address string) (*big.Int, error)
}
