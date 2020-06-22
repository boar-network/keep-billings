package billing

import (
	"fmt"
	"strings"
)

type Customer struct {
	Name            string
	Operator        string
	Beneficiary     string
	SharePercentage int
}

type Report struct {
	Customer *Customer

	NumberOfGroups           int
	NumberOfGroupMemberships int
}

type DataSource interface {
	NumberOfGroups() (int64, error)
	GroupPublicKey(index int64) ([]byte, error)
	GroupDistinctMembers(groupPublicKey []byte) (map[string]bool, error)
}

type ReportGenerator struct {
	dataSource DataSource

	groups []*group
}

type group struct {
	index     int
	publicKey []byte
	members   map[string]bool
}

func NewReportGenerator(dataSource DataSource) (*ReportGenerator, error) {
	generator := &ReportGenerator{
		dataSource: dataSource,
	}

	err := generator.fetchCommonData()
	if err != nil {
		return nil, err
	}

	return generator, nil
}

func (rg *ReportGenerator) fetchCommonData() error {
	var err error

	rg.groups, err = rg.fetchGroupsData()
	if err != nil {
		return err
	}

	// TODO: fetch keeps data

	return nil
}

func (rg *ReportGenerator) fetchGroupsData() ([]*group, error) {
	numberOfGroups, err := rg.dataSource.NumberOfGroups()
	if err != nil {
		return nil, fmt.Errorf(
			"could not get number of groups: [%v]",
			err,
		)
	}

	groups := make([]*group, numberOfGroups)
	for index := range groups {
		publicKey, err := rg.dataSource.GroupPublicKey(int64(index))
		if err != nil {
			return nil, fmt.Errorf(
				"could not get public key of group with index [%v]: [%v]",
				index,
				err,
			)
		}

		members, err := rg.dataSource.GroupDistinctMembers(publicKey)
		if err != nil {
			return nil, fmt.Errorf(
				"could not get members of group with index [%v]: [%v]",
				index,
				err,
			)
		}

		groups[index] = &group{
			index:     index,
			publicKey: publicKey,
			members:   members,
		}
	}

	return groups, nil
}

func (rg *ReportGenerator) Generate(customer *Customer) (*Report, error) {
	return &Report{
		Customer:                 customer,
		NumberOfGroups:           len(rg.groups),
		NumberOfGroupMemberships: rg.countGroupMemberships(customer.Operator),
	}, nil
}

func (rg *ReportGenerator) countGroupMemberships(operator string) int {
	count := 0

	for _, group := range rg.groups {
		if _, exists := group.members[strings.ToLower(operator)]; exists {
			count++
		}
	}

	return count
}
