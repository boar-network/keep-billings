package chain

import (
	"bytes"
	"context"
	"fmt"
	coreabi "github.com/boar-network/reports/pkg/chain/gen/core/abi"
	ecdsaabi "github.com/boar-network/reports/pkg/chain/gen/ecdsa/abi"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"log"
	"math/big"
)

type EthereumClient struct {
	client              *ethclient.Client
	operatorContract    *coreabi.KeepRandomBeaconOperatorCaller
	keepFactoryContract *ecdsaabi.BondedECDSAKeepFactoryCaller
}

func NewEthereumClient(
	url string,
	operatorContractAddress string,
	keepFactoryContractAddress string,
) (*EthereumClient, error) {
	client, err := ethclient.Dial(url)
	if err != nil {
		return nil, err
	}

	operatorContract, err := coreabi.NewKeepRandomBeaconOperatorCaller(
		common.HexToAddress(operatorContractAddress),
		client,
	)
	if err != nil {
		return nil, err
	}

	keepFactoryContract, err := ecdsaabi.NewBondedECDSAKeepFactoryCaller(
		common.HexToAddress(keepFactoryContractAddress),
		client,
	)
	if err != nil {
		return nil, err
	}

	return &EthereumClient{
		client:              client,
		operatorContract:    operatorContract,
		keepFactoryContract: keepFactoryContract,
	}, nil
}

func (ec *EthereumClient) EthBalance(address string) (*big.Int, error) {
	weiBalance, err := ec.client.BalanceAt(
		context.Background(),
		common.HexToAddress(address),
		nil,
	)
	if err != nil {
		return nil, err
	}

	return new(big.Int).Div(weiBalance, big.NewInt(1e18)), nil
}

func (ec *EthereumClient) OutboundTransactions(
	address string,
	fromBlock, toBlock int64,
) (map[int64][]string, error) {
	ctx := context.TODO()

	if fromBlock > toBlock {
		return nil, fmt.Errorf(
			"fromBlock could not be smaller than toBlock",
		)
	}

	addressFromHex := common.HexToAddress(address)

	chainID, err := ec.client.NetworkID(ctx)
	if err != nil {
		log.Fatal(err)
	}

	blocksTransactions := make(map[int64][]string)

	for blockNumber := fromBlock; blockNumber <= toBlock; blockNumber++ {
		block, err := ec.client.BlockByNumber(ctx, big.NewInt(blockNumber))
		if err != nil {
			if err == ethereum.NotFound {
				break
			}

			return nil, err
		}

		transactions := make([]string, 0)

		for _, transaction := range block.Transactions() {
			message, err := transaction.AsMessage(types.NewEIP155Signer(chainID))
			if err != nil {
				return nil, err
			}

			from := message.From()
			if !bytes.Equal(addressFromHex[:], from[:]) {
				continue
			}

			transactions = append(transactions, transaction.Hash().Hex())
		}

		blocksTransactions[blockNumber] = transactions
	}

	return blocksTransactions, nil
}

func (ec *EthereumClient) ActiveGroupsCount() (int64, error) {
	result, err := ec.operatorContract.NumberOfGroups(nil)
	if err != nil {
		return 0, err
	}

	return result.Int64(), nil
}

func (ec *EthereumClient) FirstActiveGroupIndex() (int64, error) {
	result, err := ec.operatorContract.GetFirstActiveGroupIndex(nil)
	if err != nil {
		return 0, err
	}

	return result.Int64(), nil
}

func (ec *EthereumClient) GroupPublicKey(index int64) ([]byte, error) {
	return ec.operatorContract.GetGroupPublicKey(nil, big.NewInt(index))
}

func (ec *EthereumClient) GroupMembers(
	groupPublicKey []byte,
) (map[int]string, error) {
	addresses, err := ec.operatorContract.GetGroupMembers(nil, groupPublicKey)
	if err != nil {
		return nil, err
	}

	members := make(map[int]string)
	for i, address := range addresses {
		members[i+1] = address.Hex()
	}

	return members, err
}

func (ec *EthereumClient) ActiveKeeps() (map[int64]string, error) {
	keepCount, err := ec.keepFactoryContract.GetKeepCount(nil)
	if err != nil {
		return nil, err
	}

	keeps := make(map[int64]string)

	for index := int64(0); index < keepCount.Int64(); index++ {
		address, err := ec.keepFactoryContract.GetKeepAtIndex(
			nil,
			big.NewInt(index),
		)
		if err != nil {
			return nil, err
		}

		keep, err := ec.getKeep(address.Hex())
		if err != nil {
			return nil, err
		}

		isActive, err := keep.IsActive(nil)
		if err != nil {
			return nil, err
		}

		if isActive {
			keeps[index] = address.Hex()
		}
	}

	return keeps, nil
}

func (ec *EthereumClient) KeepMembers(
	address string,
) ([]string, error) {
	keep, err := ec.getKeep(address)
	if err != nil {
		return nil, err
	}

	addresses, err := keep.GetMembers(nil)
	if err != nil {
		return nil, err
	}

	members := make([]string, 0)
	for _, address := range addresses {
		members = append(members, address.Hex())
	}

	return members, err
}

func (ec *EthereumClient) getKeep(
	address string,
) (*ecdsaabi.BondedECDSAKeepCaller, error) {
	return ecdsaabi.NewBondedECDSAKeepCaller(
		common.HexToAddress(address),
		ec.client,
	)
}
