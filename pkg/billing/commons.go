package billing

import (
	"math/big"
	"sort"

	"github.com/boar-network/keep-billings/pkg/chain"
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
	EthBalance(address string) (*big.Float, error)
	OutboundTransactions(
		address string,
		fromBlock, toBlock int64,
	) (map[int64][]string, error)
	TransactionGasPrice(hash string) (*big.Int, error)
	TransactionGasUsed(hash string) (*big.Int, error)
	TransactionMethod(hash string) (string, error)
}

type Transaction struct {
	Block     int64
	Hash      string
	Fee       string // [Gwei]
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
			fee, err := calculateTransactionFee(transactionHash, dataSource)
			if err != nil {
				return nil, err
			}

			operation, err := dataSource.TransactionMethod(transactionHash)
			if err != nil {
				return nil, err
			}

			transaction := &Transaction{
				Block:     blockNumber,
				Hash:      transactionHash,
				Fee:       fee.Text('f', 9),
				Operation: operation,
			}

			transactions = append(transactions, transaction)
		}
	}

	sort.Stable(byBlock(transactions))

	return transactions, nil
}

func calculateTransactionFee(hash string, dataSource DataSource) (*big.Float, error) {
	gasPrice, err := dataSource.TransactionGasPrice(hash)
	if err != nil {
		return nil, err
	}

	gasUsed, err := dataSource.TransactionGasUsed(hash)
	if err != nil {
		return nil, err
	}

	weiFee := new(big.Int).Mul(gasPrice, gasUsed)

	return chain.WeiToGwei(weiFee), nil
}
