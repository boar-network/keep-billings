package billing

import (
	"fmt"
	"strings"
)

type EcdsaReport struct {
	*Report

	KeepsCount            int
	KeepsMembershipsCount int
}

type EcdsaDataSource interface {
	DataSource

	KeepsCount() (int64, error)
	KeepAddress(index int64) (string, error)
	KeepDistinctMembers(address string) (map[string]bool, error)
}

type keep struct {
	index   int
	address string
	members map[string]bool
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
	keepsCount, err := erg.dataSource.KeepsCount()
	if err != nil {
		return nil, fmt.Errorf(
			"could not get keeps count: [%v]",
			err,
		)
	}

	keeps := make([]*keep, keepsCount)
	for index := range keeps {
		address, err := erg.dataSource.KeepAddress(int64(index))
		if err != nil {
			return nil, fmt.Errorf(
				"could not get address of keep with index [%v]: [%v]",
				index,
				err,
			)
		}

		members, err := erg.dataSource.KeepDistinctMembers(address)
		if err != nil {
			return nil, fmt.Errorf(
				"could not get members of keep with index [%v]: [%v]",
				index,
				err,
			)
		}

		keeps[index] = &keep{
			index:   index,
			address: address,
			members: members,
		}
	}

	return keeps, nil
}

func (erg *EcdsaReportGenerator) Generate(
	customer *Customer,
) (*EcdsaReport, error) {
	return &EcdsaReport{
		Report:                &Report{customer},
		KeepsCount:            len(erg.keeps),
		KeepsMembershipsCount: erg.countKeepsMemberships(customer.Operator),
	}, nil
}

func (erg *EcdsaReportGenerator) countKeepsMemberships(operator string) int {
	count := 0

	for _, keep := range erg.keeps {
		if _, exists := keep.members[strings.ToLower(operator)]; exists {
			count++
		}
	}

	return count
}
