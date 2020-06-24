package billing

import (
	"math/big"
	"sort"
)

type Customer struct {
	Name                    string
	Operator                string
	Beneficiary             string
	CustomerSharePercentage int
}

type Report struct {
	Customer *Customer

	OperatorBalance    string
	BeneficiaryBalance string

	AccumulatedRewards string

	FromBlock    int64
	ToBlock      int64
	Transactions []*Transaction
}

type DataSource interface {
	EthBalance(address string) (*big.Int, error)
	OutboundTransactions(
		address string,
		fromBlock, toBlock int64,
	) (map[int64][]string, error)
}

type Transaction struct {
	Block     int64
	Hash      string
	Fee       string
	Operation string
}

type byBlock []*Transaction

func (bb byBlock) Len() int {
	return len(bb)
}

func (bb byBlock) Swap(i, j int) {
	bb[i], bb[j] = bb[j], bb[i]
}

func (bb byBlock) Less(i, j int) bool {
	return bb[i].Block < bb[j].Block
}

func outboundTransactions(
	address string,
	fromBlock, toBlock int64,
	dataSource DataSource,
) ([]*Transaction, error) {
	blocksTransactions, err := dataSource.OutboundTransactions(
		address,
		fromBlock,
		toBlock,
	)
	if err != nil {
		return nil, err
	}

	transactions := make([]*Transaction, 0)

	for blockNumber, transactionsHashes := range blocksTransactions {
		for _, transactionHash := range transactionsHashes {
			transaction := &Transaction{
				Block:     blockNumber,
				Hash:      transactionHash,
				Fee:       "-",
				Operation: "-",
			}

			transactions = append(transactions, transaction)
		}
	}

	sort.Stable(byBlock(transactions))

	return transactions, nil
}
