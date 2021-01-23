package billing

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/boar-network/keep-billings/pkg/chain"
)

type EcdsaReport struct {
	*Report

	ActiveKeepsCount          int
	ActiveKeepsMembersCount   int
	ActiveKeepsSummary        []string
	InactiveKeepsMembersCount int
}

type EcdsaDataSource interface {
	DataSource

	Keeps() (map[int64]string, map[int64]string, error)
	KeepMembers(address string) ([]string, error)
	KeepMemberBalance(keepAddress, memberAddress string) (*big.Int, error)
}

type keep struct {
	index    int64
	isActive bool
	address  string
	members  []string
}

func (k *keep) hasMember(address string) bool {
	for _, member := range k.members {
		if strings.ToLower(address) == strings.ToLower(member) {
			return true
		}
	}

	return false
}

type EcdsaReportGenerator struct {
	dataSource EcdsaDataSource

	keeps []*keep
}

func NewEcdsaReportGenerator(
	dataSource EcdsaDataSource,
) *EcdsaReportGenerator {
	return &EcdsaReportGenerator{
		dataSource: dataSource,
	}
}

func (erg *EcdsaReportGenerator) FetchCommonData() error {
	var err error

	erg.keeps, err = erg.fetchKeepsData()
	if err != nil {
		return err
	}

	return nil
}

func (erg *EcdsaReportGenerator) fetchKeepsData() ([]*keep, error) {
	keeps := make([]*keep, 0)

	activeKeeps, inactiveKeeps, err := erg.dataSource.Keeps()
	if err != nil {
		return nil, fmt.Errorf("could not get keeps: [%v]", err)
	}

	for index, address := range activeKeeps {
		members, err := erg.dataSource.KeepMembers(address)
		if err != nil {
			return nil, fmt.Errorf(
				"could not get members of an active keep [%v]: [%v]",
				address,
				err,
			)
		}

		keeps = append(
			keeps,
			&keep{
				index:    index,
				isActive: true,
				address:  address,
				members:  members,
			},
		)
	}

	for index, address := range inactiveKeeps {
		members, err := erg.dataSource.KeepMembers(address)
		if err != nil {
			return nil, fmt.Errorf(
				"could not get members of inactive keep [%v]: [%v]",
				address,
				err,
			)
		}

		keeps = append(
			keeps,
			&keep{
				index:    index,
				isActive: false,
				address:  address,
				members:  members,
			},
		)
	}

	return keeps, nil
}

func (erg *EcdsaReportGenerator) Generate(
	customer *Customer,
) (*EcdsaReport, error) {
	stake, err := erg.dataSource.Stake(customer.Operator)
	if err != nil {
		return nil, err
	}

	operatorBalance, err := erg.dataSource.EthBalance(customer.Operator)
	if err != nil {
		return nil, err
	}

	beneficiaryEthBalance, err := erg.dataSource.EthBalance(customer.Beneficiary)
	if err != nil {
		return nil, err
	}

	beneficiaryKeepBalance, err := erg.dataSource.KeepBalance(customer.Beneficiary)
	if err != nil {
		return nil, err
	}

	beneficiaryTbtcBalance, err := erg.dataSource.TbtcBalance(customer.Beneficiary)
	if err != nil {
		return nil, err
	}

	accumulatedRewards, err := erg.calculateAccumulatedRewards(customer.Operator)
	if err != nil {
		return nil, err
	}

	operationalCosts := new(big.Float).Sub(
		big.NewFloat(float64(customer.InitialOperatorEthBalance)),
		operatorBalance,
	)

	baseReport := &Report{
		Customer:               customer,
		Stake:                  stake.Text('f', 0),
		OperatorBalance:        operatorBalance.Text('f', 6),
		BeneficiaryEthBalance:  beneficiaryEthBalance.Text('f', 6),
		BeneficiaryKeepBalance: beneficiaryKeepBalance.Text('f', 6),
		BeneficiaryTbtcBalance: beneficiaryTbtcBalance.Text('f', 6),
		AccumulatedRewards:     accumulatedRewards.Text('f', 6),
		OperationalCosts:       operationalCosts.Text('f', 6),
	}

	inactiveKeepMembersCount, activeKeepsSummary := erg.prepareKeepsSummary(customer.Operator)

	return &EcdsaReport{
		Report:                    baseReport,
		ActiveKeepsCount:          len(erg.keeps),
		ActiveKeepsMembersCount:   erg.countActiveKeepsMembers(customer.Operator),
		ActiveKeepsSummary:        activeKeepsSummary,
		InactiveKeepsMembersCount: inactiveKeepMembersCount,
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

func (erg *EcdsaReportGenerator) prepareKeepsSummary(
	operator string,
) (int, []string) {
	activeKeepSummary := make([]string, 0)
	inactiveKeepsMemberCount := 0

	operatorAddress := strings.ToLower(operator)

	for _, keep := range erg.keeps {
		for _, memberAddress := range keep.members {
			if operatorAddress == strings.ToLower(memberAddress) {

				if keep.isActive {
					activeKeepSummary = append(activeKeepSummary, strings.ToLower(keep.address))
				} else {
					inactiveKeepsMemberCount++
				}
			}
		}
	}

	return inactiveKeepsMemberCount, activeKeepSummary
}

func (erg *EcdsaReportGenerator) calculateAccumulatedRewards(
	operator string,
) (*big.Float, error) {
	accumulatedRewardsWei := big.NewInt(0)

	for _, keep := range erg.keeps {
		if !keep.hasMember(operator) {
			continue
		}

		keepMemberBalanceWei, err := erg.dataSource.KeepMemberBalance(
			keep.address,
			operator,
		)
		if err != nil {
			return nil, err
		}

		accumulatedRewardsWei = new(big.Int).Add(
			accumulatedRewardsWei,
			keepMemberBalanceWei,
		)
	}

	return chain.WeiToEth(accumulatedRewardsWei), nil
}
