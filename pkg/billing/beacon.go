package billing

import (
	"fmt"
	"strings"
)

type BeaconReport struct {
	*Report

	ActiveGroupsCount        int
	ActiveGroupsMembersCount int
}

type BeaconDataSource interface {
	DataSource

	ActiveGroupsCount() (int64, error)
	FirstActiveGroupIndex() (int64, error)
	GroupPublicKey(index int64) ([]byte, error)
	GroupMembers(groupPublicKey []byte) (map[int]string, error)
}

type group struct {
	index     int64
	publicKey []byte
	members   map[int]string
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
	activeGroupsCount, err := brg.dataSource.ActiveGroupsCount()
	if err != nil {
		return nil, fmt.Errorf(
			"could not get groups count: [%v]",
			err,
		)
	}

	firstActiveGroupIndex, err := brg.dataSource.FirstActiveGroupIndex()
	if err != nil {
		return nil, fmt.Errorf(
			"could not get first active group index: [%v]",
			err,
		)
	}

	groups := make([]*group, 0)

	// TODO: resolve terminated groups issue:
	//  - activeGroupsCount is the number of active groups and doesn't
	//    count terminated ones
	//  - firstActiveGroupIndex is just the number of expired groups
	//  A problematic scenario:
	//  We have 5 groups with indexes: 0,1,2,3,4.
	//  Suppose group 0 is expired and group 3 is terminated.
	//  So, firstActiveGroupIndex is 1 and activeGroupsCount is 3.
	//  Because of that we will iterate on 1,2,3 instead of 1,2,4.
	for i := int64(0); i < activeGroupsCount; i++ {
		index := firstActiveGroupIndex + i

		publicKey, err := brg.dataSource.GroupPublicKey(index)
		if err != nil {
			return nil, fmt.Errorf(
				"could not get public key of group with index [%v]: [%v]",
				index,
				err,
			)
		}

		members, err := brg.dataSource.GroupMembers(publicKey)
		if err != nil {
			return nil, fmt.Errorf(
				"could not get members of group with index [%v]: [%v]",
				index,
				err,
			)
		}

		groups = append(
			groups,
			&group{
				index:     index,
				publicKey: publicKey,
				members:   members,
			},
		)
	}

	return groups, nil
}

func (brg *BeaconReportGenerator) Generate(
	customer *Customer,
) (*BeaconReport, error) {
	return &BeaconReport{
		Report:                   &Report{customer},
		ActiveGroupsCount:        len(brg.groups),
		ActiveGroupsMembersCount: brg.countActiveGroupsMembers(customer.Operator),
	}, nil
}

func (brg *BeaconReportGenerator) countActiveGroupsMembers(operator string) int {
	count := 0

	operatorAddress := strings.ToLower(operator)

	for _, group := range brg.groups {
		for _, memberAddress := range group.members {
			if operatorAddress == strings.ToLower(memberAddress) {
				count++
			}
		}
	}

	return count
}
