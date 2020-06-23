package chain

import (
	coreabi "github.com/boar-network/reports/pkg/chain/gen/core/abi"
	ecdsaabi "github.com/boar-network/reports/pkg/chain/gen/ecdsa/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
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

func (ec *EthereumClient) GetBalance(address string) (*big.Int, error) {
	// TODO: implementation.
	return big.NewInt(0), nil
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

func (ec *EthereumClient) KeepsCount() (int64, error) {
	result, err := ec.keepFactoryContract.GetKeepCount(nil)
	if err != nil {
		return 0, err
	}

	return result.Int64(), nil
}

func (ec *EthereumClient) KeepAddress(index int64) (string, error) {
	address, err := ec.keepFactoryContract.GetKeepAtIndex(nil, big.NewInt(index))
	if err != nil {
		return "", err
	}

	return address.Hex(), err
}

func (ec *EthereumClient) KeepDistinctMembers(
	address string,
) (map[string]bool, error) {
	keep, err := ec.getKeep(address)
	if err != nil {
		return nil, err
	}

	addresses, err := keep.GetMembers(nil)
	if err != nil {
		return nil, err
	}

	hexes := make(map[string]bool)
	for _, address := range addresses {
		hexes[address.Hex()] = true
	}

	return hexes, err
}

func (ec *EthereumClient) getKeep(
	address string,
) (*ecdsaabi.BondedECDSAKeepCaller, error) {
	return ecdsaabi.NewBondedECDSAKeepCaller(
		common.HexToAddress(address),
		ec.client,
	)
}
