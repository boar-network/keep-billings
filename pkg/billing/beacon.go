package billing

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"sort"
	"strings"

	"github.com/boar-network/keep-billings/pkg/chain"
)

type BeaconReport struct {
	*Report

	ActiveGroupsCount        int
	ActiveGroupsMembersCount int
	GroupsSummary            map[string]string
}

type BeaconDataSource interface {
	DataSource

	ActiveGroupsCount() (int64, error)
	FirstActiveGroupIndex() (int64, error)
	GroupPublicKey(index int64) ([]byte, error)
	GroupMembers(groupPublicKey []byte) (map[int]string, error)
	GroupMemberRewards(groupPublicKey []byte) (*big.Int, error)
	AreRewardsWithdrawn(operator string, groupIndex int64) (bool, error)
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
			"could not get active groups count: [%v]",
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
	//
	// The new version of keep random beacon operator contract has
	// getNumberOfCreatedGroups function.
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
	fromBlock, toBlock int64,
) (*BeaconReport, error) {
	operatorBalance, err := brg.dataSource.EthBalance(customer.Operator)
	if err != nil {
		return nil, err
	}

	beneficiaryBalance, err := brg.dataSource.EthBalance(customer.Beneficiary)
	if err != nil {
		return nil, err
	}

	accumulatedRewards, err := brg.calculateAccumulatedRewards(customer.Operator)
	if err != nil {
		return nil, err
	}

	transactions, err := outboundTransactions(
		customer.Operator,
		fromBlock,
		toBlock,
		brg.dataSource,
	)
	if err != nil {
		return nil, err
	}

	baseReport := &Report{
		Customer:           customer,
		OperatorBalance:    operatorBalance.Text('f', 6),
		BeneficiaryBalance: beneficiaryBalance.Text('f', 6),
		AccumulatedRewards: accumulatedRewards.Text('f', 6),
		FromBlock:          fromBlock,
		ToBlock:            toBlock,
		Transactions:       transactions,
	}

	return &BeaconReport{
		Report:                   baseReport,
		ActiveGroupsCount:        len(brg.groups),
		ActiveGroupsMembersCount: brg.countActiveGroupsMembers(customer.Operator),
		GroupsSummary:            brg.prepareGroupsSummary(customer.Operator),
	}, nil
}

func (brg *BeaconReportGenerator) countActiveGroupsMembers(operator string) int {
	count := 0

	for _, group := range brg.groups {
		count += countActiveGroupMembers(operator, group)
	}

	return count
}

func countActiveGroupMembers(
	operator string,
	group *group,
) int {
	count := 0

	operatorAddress := strings.ToLower(operator)

	for _, memberAddress := range group.members {
		if operatorAddress == strings.ToLower(memberAddress) {
			count++
		}
	}

	return count
}

func (brg *BeaconReportGenerator) prepareGroupsSummary(
	operator string,
) map[string]string {
	groupsSummary := make(map[string]string)

	operatorAddress := strings.ToLower(operator)

	for _, group := range brg.groups {
		operatorMembers := make([]int, 0)

		for memberIndex, memberAddress := range group.members {
			if operatorAddress == strings.ToLower(memberAddress) {
				operatorMembers = append(operatorMembers, memberIndex)
			}
		}

		sort.Ints(operatorMembers)

		operatorMembersString := strings.Trim(
			strings.Join(
				strings.Fields(fmt.Sprint(operatorMembers)),
				", ",
			),
			"[]",
		)

		if len(operatorMembersString) == 0 {
			operatorMembersString = "No members"
		}

		group := "0x" + hex.EncodeToString(group.publicKey)[:32] + "..."

		groupsSummary[group] = operatorMembersString
	}

	return groupsSummary
}

func (brg *BeaconReportGenerator) calculateAccumulatedRewards(
	operator string,
) (*big.Float, error) {
	accumulatedRewardsWei := big.NewInt(0)

	for _, group := range brg.groups {
		rewardsWithdrawn, err := brg.dataSource.AreRewardsWithdrawn(
			operator,
			group.index,
		)
		if err != nil {
			return nil, err
		}

		if rewardsWithdrawn {
			continue
		}

		memberRewards, err := brg.dataSource.GroupMemberRewards(group.publicKey)
		if err != nil {
			return nil, err
		}

		groupRewardsWei := new(big.Int).Mul(
			memberRewards,
			big.NewInt(int64(countActiveGroupMembers(operator, group))),
		)

		accumulatedRewardsWei = new(big.Int).Add(
			accumulatedRewardsWei,
			groupRewardsWei,
		)
	}

	return chain.WeiToEth(accumulatedRewardsWei), nil
}
