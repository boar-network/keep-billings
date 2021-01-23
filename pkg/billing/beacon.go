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

	TotalGroupsCount           int
	ActiveGroupsCount          int
	ActiveGroupsMembersCount   int
	ActiveGroupsSummary        map[string]string
	InactiveGroupsMembersCount int
}

type BeaconDataSource interface {
	DataSource

	AllGroupsCount() (int64, error)
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
	numberOfAllGroups, err := brg.dataSource.AllGroupsCount()
	if err != nil {
		return nil, fmt.Errorf(
			"could not get total group count: [%v]",
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
) (*BeaconReport, error) {
	stake, err := brg.dataSource.Stake(customer.Operator)
	if err != nil {
		return nil, err
	}

	operatorEthBalance, err := brg.dataSource.EthBalance(customer.Operator)
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

	accumulatedEthRewards, err := brg.calculateAccumulatedRewards(customer.Operator)
	if err != nil {
		return nil, err
	}

	customerEthRewardsShare, providerEthRewardsShare,
		customerKeepRewardsShare, providerKeepRewardsShare :=
		calculateFinalBeaconRewards(
			big.NewFloat(float64(customer.CustomerSharePercentage)),
			beneficiaryEthBalance,
			beneficiaryKeepBalance,
			accumulatedEthRewards,
		)

	baseReport := &Report{
		Customer:               customer,
		Stake:                  stake.Text('f', 0),
		OperatorBalance:        operatorEthBalance.Text('f', 6),
		BeneficiaryEthBalance:  beneficiaryEthBalance.Text('f', 6),
		BeneficiaryKeepBalance: beneficiaryKeepBalance.Text('f', 6),
		AccumulatedRewards:     accumulatedEthRewards.Text('f', 6),
		CustomerEthShare:       customerEthRewardsShare.Text('f', 6),
		ProviderEthShare:       providerEthRewardsShare.Text('f', 6),
		CustomerKeepShare:      customerKeepRewardsShare.Text('f', 6),
		ProviderKeepShare:      providerKeepRewardsShare.Text('f', 6),
	}

	activeGroupsMemberCount, inactiveGroupsMemberCount,
		activeGroupsSummary := brg.summarizeGroupsInfo(
		customer.Operator,
	)

	return &BeaconReport{
		Report:                     baseReport,
		TotalGroupsCount:           len(brg.groups),
		ActiveGroupsCount:          len(activeGroupsSummary),
		ActiveGroupsMembersCount:   activeGroupsMemberCount,
		ActiveGroupsSummary:        activeGroupsSummary,
		InactiveGroupsMembersCount: inactiveGroupsMemberCount,
	}, nil
}

func (brg *BeaconReportGenerator) summarizeGroupsInfo(
	operator string,
) (
	// count of members for the operator in active groups
	activeGroupsMemberCount int,
	// count of members for the operator in no longer active groups
	inactiveGroupsMemberCount int,
	// summary of all active groups, no matter if the operator has a member
	// in a group or not
	activeGroupsSummary map[string]string,
) {
	activeGroupsMemberCount = 0
	inactiveGroupsMemberCount = 0
	activeGroupsSummary = make(map[string]string)

	for _, group := range brg.groups {
		operatorMembers := getGroupMemberIndexes(operator, group)

		if !group.isActive {
			inactiveGroupsMemberCount += len(operatorMembers)
			continue
		}
		activeGroupsMemberCount += len(operatorMembers)

		sort.Ints(operatorMembers)

		operatorMembersString := strings.Trim(
			strings.Join(
				strings.Fields(fmt.Sprint(operatorMembers)),
				", ",
			),
			"[]",
		)

		if len(operatorMembersString) == 0 {
			operatorMembersString = "-"
		}

		group := "0x" + hex.EncodeToString(group.publicKey)[:32] + "..."

		activeGroupsSummary[group] = operatorMembersString
	}

	return
}

func getGroupMemberIndexes(operatorAddress string, _group *group) []int {
	operatorMembers := make([]int, 0)

	for memberIndex, memberAddress := range _group.members {
		if strings.ToLower(operatorAddress) == strings.ToLower(memberAddress) {
			operatorMembers = append(operatorMembers, memberIndex)
		}
	}

	return operatorMembers
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

		operatorMembers := getGroupMemberIndexes(operator, group)

		groupRewardsWei := new(big.Int).Mul(
			memberRewards,
			big.NewInt(int64(len(operatorMembers))),
		)

		accumulatedRewardsWei = new(big.Int).Add(
			accumulatedRewardsWei,
			groupRewardsWei,
		)
	}

	return chain.WeiToEth(accumulatedRewardsWei), nil
}

func calculateFinalBeaconRewards(
	customerSharePercentage *big.Float,
	beneficiaryEthBalance *big.Float,
	beneficiaryKeepBalance *big.Float,
	accumulatedEthRewards *big.Float,
) (
	customerEthRewardShare *big.Float,
	providerEthRewardShare *big.Float,
	customerKeepRewardShare *big.Float,
	providerKeepRewardShare *big.Float,
) {
	customerKeepRewardShare = new(big.Float).Quo(
		new(big.Float).Mul(beneficiaryKeepBalance, customerSharePercentage),
		big.NewFloat(100),
	)
	providerKeepRewardShare = new(big.Float).Sub(
		beneficiaryKeepBalance,
		customerKeepRewardShare,
	)

	customerAccumulatedEthRewardShare := new(big.Float).Quo(
		new(big.Float).Mul(accumulatedEthRewards, customerSharePercentage),
		big.NewFloat(100),
	)

	customerEthRewardShare = new(big.Float).Add(
		customerAccumulatedEthRewardShare, beneficiaryEthBalance,
	)

	providerEthRewardShare = new(big.Float).Sub(
		accumulatedEthRewards,
		customerAccumulatedEthRewardShare,
	)

	return
}
