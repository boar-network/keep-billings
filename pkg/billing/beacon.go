package billing

import (
	"fmt"
	"strings"
)

type BeaconReport struct {
	*Report

	GroupsCount            int
	GroupsMembershipsCount int
}

type BeaconDataSource interface {
	DataSource

	GroupsCount() (int64, error)
	GroupPublicKey(index int64) ([]byte, error)
	GroupDistinctMembers(groupPublicKey []byte) (map[string]bool, error)
}

type group struct {
	index     int
	publicKey []byte
	members   map[string]bool
}

type BeaconReportGenerator struct {
	dataSource BeaconDataSource

	groups []*group
}

func NewBeaconReportGenerator(
	dataSource BeaconDataSource,
) (*BeaconReportGenerator, error) {
	generator := &BeaconReportGenerator{
		dataSource: dataSource,
	}

	err := generator.fetchCommonData()
	if err != nil {
		return nil, err
	}

	return generator, nil
}

func (brg *BeaconReportGenerator) fetchCommonData() error {
	var err error

	brg.groups, err = brg.fetchGroupsData()
	if err != nil {
		return err
	}

	return nil
}

func (brg *BeaconReportGenerator) fetchGroupsData() ([]*group, error) {
	groupsCount, err := brg.dataSource.GroupsCount()
	if err != nil {
		return nil, fmt.Errorf(
			"could not get groups count: [%v]",
			err,
		)
	}

	groups := make([]*group, groupsCount)
	for index := range groups {
		publicKey, err := brg.dataSource.GroupPublicKey(int64(index))
		if err != nil {
			return nil, fmt.Errorf(
				"could not get public key of group with index [%v]: [%v]",
				index,
				err,
			)
		}

		members, err := brg.dataSource.GroupDistinctMembers(publicKey)
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

func (brg *BeaconReportGenerator) Generate(
	customer *Customer,
) (*BeaconReport, error) {
	return &BeaconReport{
		Report:                 &Report{customer},
		GroupsCount:            len(brg.groups),
		GroupsMembershipsCount: brg.countGroupsMemberships(customer.Operator),
	}, nil
}

func (brg *BeaconReportGenerator) countGroupsMemberships(operator string) int {
	count := 0

	for _, group := range brg.groups {
		if _, exists := group.members[strings.ToLower(operator)]; exists {
			count++
		}
	}

	return count
}
