package billing

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/boar-network/keep-billings/pkg/chain"
)

type EcdsaReport struct {
	*Report

	ActiveKeepsCount        int
	ActiveKeepsMembersCount int
	KeepsSummary            []string
}

type EcdsaDataSource interface {
	DataSource

	ActiveKeeps() (map[int64]string, error)
	KeepMembers(address string) ([]string, error)
	KeepMemberBalance(keepAddress, memberAddress string) (*big.Int, error)
}

type keep struct {
	index   int64
	address string
	members []string
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
	fromBlock, toBlock int64,
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

	accumulatedRewards, err := erg.calculateAccumulatedRewards(customer.Operator)
	if err != nil {
		return nil, err
	}

	transactions, err := outboundTransactions(
		customer.Operator,
		fromBlock,
		toBlock,
		erg.dataSource,
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
		FromBlock:              fromBlock,
		ToBlock:                toBlock,
		Transactions:           transactions,
	}

	return &EcdsaReport{
		Report:                  baseReport,
		ActiveKeepsCount:        len(erg.keeps),
		ActiveKeepsMembersCount: erg.countActiveKeepsMembers(customer.Operator),
		KeepsSummary:            erg.prepareKeepsSummary(customer.Operator),
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
) []string {
	keepSummary := make([]string, 0)

	operatorAddress := strings.ToLower(operator)

	for _, keep := range erg.keeps {
		for _, memberAddress := range keep.members {
			if operatorAddress == strings.ToLower(memberAddress) {
				keepSummary = append(keepSummary, strings.ToLower(keep.address))
			}
		}
	}

	return keepSummary
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
