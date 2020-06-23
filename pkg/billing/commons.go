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
}

type DataSource interface {
	GetBalance(address string) (*big.Int, error)
}
