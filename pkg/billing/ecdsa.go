package billing

import (
	"fmt"
	"strings"
)

type EcdsaReport struct {
	*Report

	ActiveKeepsCount        int
	ActiveKeepsMembersCount int
}

type EcdsaDataSource interface {
	DataSource

	ActiveKeeps() (map[int64]string, error)
	KeepMembers(address string) ([]string, error)
}

type keep struct {
	index   int64
	address string
	members []string
}

type EcdsaReportGenerator struct {
	dataSource EcdsaDataSource

	keeps []*keep
}

func NewEcdsaReportGenerator(
	dataSource EcdsaDataSource,
) (*EcdsaReportGenerator, error) {
	generator := &EcdsaReportGenerator{
		dataSource: dataSource,
	}

	err := generator.fetchCommonData()
	if err != nil {
		return nil, err
	}

	return generator, nil
}

func (erg *EcdsaReportGenerator) fetchCommonData() error {
	var err error

	erg.keeps, err = erg.fetchKeepsData()
	if err != nil {
		return err
	}

	return nil
}

func (erg *EcdsaReportGenerator) fetchKeepsData() ([]*keep, error) {
	keeps := make([]*keep, 0)

	activeKeeps, err := erg.dataSource.ActiveKeeps()
	if err != nil {
		return nil, fmt.Errorf("could not get active keeps: [%v]", err)
	}

	for index, address := range activeKeeps {
		members, err := erg.dataSource.KeepMembers(address)
		if err != nil {
			return nil, fmt.Errorf(
				"could not get members of keep [%v]: [%v]",
				address,
				err,
			)
		}

		keeps = append(
			keeps,
			&keep{
				index:   index,
				address: address,
				members: members,
			},
		)
	}

	return keeps, nil
}

func (erg *EcdsaReportGenerator) Generate(
	customer *Customer,
) (*EcdsaReport, error) {
	operatorBalance, err := erg.dataSource.GetEthBalance(customer.Operator)
	if err != nil {
		return nil, err
	}

	beneficiaryBalance, err := erg.dataSource.GetEthBalance(customer.Beneficiary)
	if err != nil {
		return nil, err
	}

	baseReport := &Report{
		Customer:           customer,
		OperatorBalance:    operatorBalance.String(),
		BeneficiaryBalance: beneficiaryBalance.String(),
		AccumulatedRewards: "-",
	}

	return &EcdsaReport{
		Report:                  baseReport,
		ActiveKeepsCount:        len(erg.keeps),
		ActiveKeepsMembersCount: erg.countActiveKeepsMembers(customer.Operator),
	}, nil
}

func (erg *EcdsaReportGenerator) countActiveKeepsMembers(operator string) int {
	count := 0

	operatorAddress := strings.ToLower(operator)

	for _, keep := range erg.keeps {
		for _, memberAddress := range keep.members {
			if operatorAddress == strings.ToLower(memberAddress) {
				count++
			}
		}
	}

	return count
}
