package billing

import (
	"fmt"
	"strings"
)

type Customer struct {
	Name                    string
	Operator                string
	Beneficiary             string
	CustomerSharePercentage int
}

type Report struct {
	Customer *Customer

	GroupsCount            int
	GroupsMembershipsCount int
	KeepsCount             int
	KeepsMembershipsCount  int
}

type DataSource interface {
	GroupsCount() (int64, error)
	GroupPublicKey(index int64) ([]byte, error)
	GroupDistinctMembers(groupPublicKey []byte) (map[string]bool, error)

	KeepsCount() (int64, error)
	KeepAddress(index int64) (string, error)
	KeepDistinctMembers(address string) (map[string]bool, error)
}

type ReportGenerator struct {
	dataSource DataSource

	groups []*group
	keeps  []*keep
}

type group struct {
	index     int
	publicKey []byte
	members   map[string]bool
}

type keep struct {
	index   int
	address string
	members map[string]bool
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

	rg.keeps, err = rg.fetchKeepsData()
	if err != nil {
		return err
	}

	return nil
}

func (rg *ReportGenerator) fetchGroupsData() ([]*group, error) {
	groupsCount, err := rg.dataSource.GroupsCount()
	if err != nil {
		return nil, fmt.Errorf(
			"could not get groups count: [%v]",
			err,
		)
	}

	groups := make([]*group, groupsCount)
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

func (rg *ReportGenerator) fetchKeepsData() ([]*keep, error) {
	keepsCount, err := rg.dataSource.KeepsCount()
	if err != nil {
		return nil, fmt.Errorf(
			"could not get keeps count: [%v]",
			err,
		)
	}

	keeps := make([]*keep, keepsCount)
	for index := range keeps {
		address, err := rg.dataSource.KeepAddress(int64(index))
		if err != nil {
			return nil, fmt.Errorf(
				"could not get address of keep with index [%v]: [%v]",
				index,
				err,
			)
		}

		members, err := rg.dataSource.KeepDistinctMembers(address)
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

func (rg *ReportGenerator) Generate(customer *Customer) (*Report, error) {
	return &Report{
		Customer:               customer,
		GroupsCount:            len(rg.groups),
		GroupsMembershipsCount: rg.countGroupsMemberships(customer.Operator),
		KeepsCount:             len(rg.keeps),
		KeepsMembershipsCount:  rg.countKeepsMemberships(customer.Operator),
	}, nil
}

func (rg *ReportGenerator) countGroupsMemberships(operator string) int {
	count := 0

	for _, group := range rg.groups {
		if _, exists := group.members[strings.ToLower(operator)]; exists {
			count++
		}
	}

	return count
}

func (rg *ReportGenerator) countKeepsMemberships(operator string) int {
	count := 0

	for _, keep := range rg.keeps {
		if _, exists := keep.members[strings.ToLower(operator)]; exists {
			count++
		}
	}

	return count
}
