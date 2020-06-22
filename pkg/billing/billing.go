package billing

type Customer struct {
	Name            string
	Operator        string
	Beneficiary     string
	SharePercentage int
}

type Report struct {
	Customer     string
	Transactions int
}

type DataSource interface {
	NumberOfGroups() (int64, error)
}

func GenerateReport(customer *Customer, data DataSource) (*Report, error) {
	// TODO: implementation
	report := &Report{
		Customer:     customer.Name,
		Transactions: len(customer.Name),
	}

	return report, nil
}
