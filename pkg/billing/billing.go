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

func GenerateReport(customer *Customer) (*Report, error) {
	// TODO: implementation
	report := &Report{
		Customer:     customer.Name,
		Transactions: len(customer.Name),
	}

	return report, nil
}
