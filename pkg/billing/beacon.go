package billing

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"sort"
	"strings"

	"github.com/boar-network/keep-billings/pkg/chain"
	"github.com/ethereum/go-ethereum/core/types"
)

type BeaconReport struct {
	*Report

	ActiveGroupsCount          int
	ActiveGroupsMembersCount   int
	ActiveGroupsSummary        map[string]string
	InactiveGroupsMembersCount int
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
	isActive  bool
	publicKey []byte
	members   map[int]string
}

type BeaconReportGenerator struct {
	dataSource BeaconDataSource

	groups []*group
}

func NewBeaconReportGenerator(
	dataSource BeaconDataSource,
) *BeaconReportGenerator {
	return &BeaconReportGenerator{
		dataSource: dataSource,
	}
}

func (brg *BeaconReportGenerator) FetchCommonData() error {
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
	numberOfAllGroups := firstActiveGroupIndex + activeGroupsCount

	for index := int64(0); index < numberOfAllGroups; index++ {
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

		isActive := false
		if index >= firstActiveGroupIndex {
			isActive = true
		}

		groups = append(
			groups,
			&group{
				index:     index,
				isActive:  isActive,
				publicKey: publicKey,
				members:   members,
			},
		)
	}

	return groups, nil
}

func (brg *BeaconReportGenerator) Generate(
	customer *Customer,
	blocks []*types.Block,
) (*BeaconReport, error) {
	stake, err := brg.dataSource.Stake(customer.Operator)
	if err != nil {
		return nil, err
	}

	operatorBalance, err := brg.dataSource.EthBalance(customer.Operator)
	if err != nil {
		return nil, err
	}

	beneficiaryEthBalance, err := brg.dataSource.EthBalance(customer.Beneficiary)
	if err != nil {
		return nil, err
	}

	beneficiaryKeepBalance, err := brg.dataSource.KeepBalance(customer.Beneficiary)
	if err != nil {
		return nil, err
	}

	accumulatedRewards, err := brg.calculateAccumulatedRewards(customer.Operator)
	if err != nil {
		return nil, err
	}

	transactions, err := outboundTransactions(
		customer.Operator,
		blocks,
		brg.dataSource,
	)
	if err != nil {
		return nil, err
	}

	baseReport := &Report{
		Customer:               customer,
		Stake:                  stake.Text('f', 0),
		OperatorBalance:        operatorBalance.Text('f', 6),
		BeneficiaryEthBalance:  beneficiaryEthBalance.Text('f', 6),
		BeneficiaryKeepBalance: beneficiaryKeepBalance.Text('f', 6),
		AccumulatedRewards:     accumulatedRewards.Text('f', 6),
		FromBlock:              blocks[0].NumberU64(),
		ToBlock:                blocks[len(blocks)-1].NumberU64(),
		Transactions:           transactions,
	}

	inactiveGroupsMemberCount, activeGroupsSummary := brg.summarizeGroupsInfo(
		customer.Operator,
	)

	return &BeaconReport{
		Report:                     baseReport,
		ActiveGroupsCount:          len(brg.groups),
		ActiveGroupsMembersCount:   brg.countActiveGroupsMembers(customer.Operator),
		ActiveGroupsSummary:        activeGroupsSummary,
		InactiveGroupsMembersCount: inactiveGroupsMemberCount,
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

func (brg *BeaconReportGenerator) summarizeGroupsInfo(
	operator string,
) (int, map[string]string) {
	inactiveGroupsMemberCount := 0
	activeGroupsSummary := make(map[string]string)

	operatorAddress := strings.ToLower(operator)

	for _, group := range brg.groups {
		operatorMembers := make([]int, 0)

		for memberIndex, memberAddress := range group.members {
			if operatorAddress == strings.ToLower(memberAddress) {
				operatorMembers = append(operatorMembers, memberIndex)
			}
		}

		if !group.isActive {
			inactiveGroupsMemberCount += len(operatorMembers)
			continue
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

		activeGroupsSummary[group] = operatorMembersString
	}

	return inactiveGroupsMemberCount, activeGroupsSummary
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
