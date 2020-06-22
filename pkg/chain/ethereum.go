package chain

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/keep-network/keep-core/pkg/chain/gen/abi"
)

type EthereumClient struct {
	client       *ethclient.Client
	krboContract *abi.KeepRandomBeaconOperatorCaller
}

func NewEthereumClient(
	url string,
	krboContractAddress string,
) (*EthereumClient, error) {
	client, err := ethclient.Dial(url)
	if err != nil {
		return nil, err
	}

	krboContract, err := abi.NewKeepRandomBeaconOperatorCaller(
		common.HexToAddress(krboContractAddress),
		client,
	)
	if err != nil {
		return nil, err
	}

	return &EthereumClient{
		client:       client,
		krboContract: krboContract,
	}, nil
}

func (ec *EthereumClient) NumberOfGroups() (int64, error) {
	result, err := ec.krboContract.NumberOfGroups(nil)
	if err != nil {
		return 0, err
	}

	return result.Int64(), nil
}
