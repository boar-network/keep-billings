package billing

import (
	"math/big"
	"sort"

	"github.com/ipfs/go-log"

	"github.com/boar-network/keep-billings/pkg/chain"
)

var logger = log.Logger("billings-billing")

type Customer struct {
	Name                    string
	Operator                string
	Beneficiary             string
	CustomerSharePercentage int
}

type Report struct {
	Customer *Customer

	Stake                  string
	OperatorBalance        string
	BeneficiaryEthBalance  string
	BeneficiaryKeepBalance string
	BeneficiaryTbtcBalance string

	AccumulatedRewards string

	FromBlock    uint64
	ToBlock      uint64
	Transactions []*Transaction
}

type DataSource interface {
	EthBalance(address string) (*big.Float, error)
	Stake(address string) (*big.Float, error)
	KeepBalance(address string) (*big.Float, error)
	TbtcBalance(address string) (*big.Float, error)
	TransactionGasPrice(hash string) (*big.Int, error)
	TransactionGasUsed(hash string) (*big.Int, error)
	TransactionMethod(hash string) (string, error)
}

type CachedBlocks interface {
	FirstBlockNumber() uint64
	LastBlockNumber() uint64
	FilterOutboundTransactions(address string) (map[int64][]string, error)
}

type Transaction struct {
	Block     int64
	Hash      string
	Fee       string // [Gwei]
	Gas       string
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
	blocks CachedBlocks,
	dataSource DataSource,
) ([]*Transaction, error) {
	logger.Infof(
		"filtering out outbound transactions for address [%v]",
		address,
	)
	blocksTransactions, err := blocks.FilterOutboundTransactions(address)
	if err != nil {
		return nil, err
	}
	logger.Infof("outbound transactions filtered out")

	transactions := make([]*Transaction, 0)

	for blockNumber, transactionsHashes := range blocksTransactions {
		for _, transactionHash := range transactionsHashes {
			gasUsed, err := dataSource.TransactionGasUsed(transactionHash)
			if err != nil {
				return nil, err
			}

			fee, err := calculateTransactionFee(
				transactionHash,
				gasUsed,
				dataSource,
			)
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
				Gas:       gasUsed.Text(10),
				Operation: operation,
			}

			transactions = append(transactions, transaction)
		}
	}

	sort.Stable(byBlock(transactions))

	return transactions, nil
}

func calculateTransactionFee(
	hash string,
	gasUsed *big.Int,
	dataSource DataSource,
) (*big.Float, error) {
	gasPrice, err := dataSource.TransactionGasPrice(hash)
	if err != nil {
		return nil, err
	}

	weiFee := new(big.Int).Mul(gasPrice, gasUsed)

	return chain.WeiToGwei(weiFee), nil
}
