package chain

import (
	"bytes"
	"context"
	"fmt"
	"math"
	"math/big"
	"strings"

	"github.com/ipfs/go-log"

	coreabi "github.com/boar-network/keep-billings/pkg/chain/gen/core/abi"
	ecdsaabi "github.com/boar-network/keep-billings/pkg/chain/gen/ecdsa/abi"
	erc20abi "github.com/boar-network/keep-billings/pkg/chain/gen/erc20/abi"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

var logger = log.Logger("billings-ethereum")

var methodLookupAbiStrings = []string{
	coreabi.TokenStakingABI,
	coreabi.KeepRandomBeaconOperatorABI,
	coreabi.KeepRandomBeaconServiceImplV1ABI,
	ecdsaabi.BondedECDSAKeepFactoryABI,
	ecdsaabi.BondedECDSAKeepABI,
	ecdsaabi.KeepBondingABI,
}

type EthereumClient struct {
	client              *ethclient.Client
	keepToken           *erc20abi.TokenCaller
	tbtcToken           *erc20abi.TokenCaller
	tokenStaking        *coreabi.TokenStakingCaller
	operatorContract    *coreabi.KeepRandomBeaconOperatorCaller
	keepFactoryContract *ecdsaabi.BondedECDSAKeepFactoryCaller
	methodLookupAbiList []abi.ABI
}

func NewEthereumClient(
	url string,
	keepTokenAddress string,
	tbtcTokenAddress string,
	tokenStakingAddress string,
	operatorContractAddress string,
	keepFactoryContractAddress string,
) (*EthereumClient, error) {
	client, err := ethclient.Dial(url)
	if err != nil {
		return nil, err
	}

	keepToken, err := erc20abi.NewTokenCaller(
		common.HexToAddress(keepTokenAddress),
		client,
	)
	if err != nil {
		return nil, err
	}

	tbtcToken, err := erc20abi.NewTokenCaller(
		common.HexToAddress(tbtcTokenAddress),
		client,
	)
	if err != nil {
		return nil, err
	}

	tokenStaking, err := coreabi.NewTokenStakingCaller(
		common.HexToAddress(tokenStakingAddress),
		client,
	)
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

	methodLookupAbiList := make([]abi.ABI, len(methodLookupAbiStrings))
	for i := range methodLookupAbiList {
		methodLookupAbiList[i], err = abi.JSON(
			strings.NewReader(methodLookupAbiStrings[i]),
		)
		if err != nil {
			return nil, err
		}
	}

	return &EthereumClient{
		client:              client,
		keepToken:           keepToken,
		tbtcToken:           tbtcToken,
		tokenStaking:        tokenStaking,
		operatorContract:    operatorContract,
		keepFactoryContract: keepFactoryContract,
		methodLookupAbiList: methodLookupAbiList,
	}, nil
}

func (ec *EthereumClient) KeepBalance(address string) (*big.Float, error) {
	balance, err := ec.keepToken.BalanceOf(nil, common.HexToAddress(address))
	if err != nil {
		return nil, err
	}

	// it's not ETH but KEEP ERC-20 uses the same number of decimals
	return WeiToEth(balance), nil
}

func (ec *EthereumClient) TbtcBalance(address string) (*big.Float, error) {
	balance, err := ec.tbtcToken.BalanceOf(nil, common.HexToAddress(address))
	if err != nil {
		return nil, err
	}

	// it's not ETH but tBTC ERC-20 uses the same number of decimals
	return WeiToEth(balance), nil
}

func (ec *EthereumClient) EthBalance(address string) (*big.Float, error) {
	weiBalance, err := ec.client.BalanceAt(
		context.Background(),
		common.HexToAddress(address),
		nil,
	)
	if err != nil {
		return nil, err
	}

	return WeiToEth(weiBalance), nil
}

func (ec *EthereumClient) Stake(address string) (*big.Float, error) {
	stake, err := ec.tokenStaking.BalanceOf(nil, common.HexToAddress(address))
	if err != nil {
		return nil, err
	}

	// it's not ETH but KEEP ERC-20 uses the same number of decimals
	return WeiToEth(stake), nil
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
		logger.Fatal(err)
	}

	blocksTransactions := make(map[int64][]string)

	for blockNumber := fromBlock; blockNumber <= toBlock; blockNumber++ {
		progress := math.Floor((float64(blockNumber-fromBlock) / float64(toBlock-fromBlock)) * 100)
		logger.Infof("[%.0f%%] getting block [%v]", progress, blockNumber)
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

func (ec *EthereumClient) TransactionGasPrice(txHash string) (*big.Int, error) {
	ctx := context.TODO()

	transaction, _, err := ec.client.TransactionByHash(
		ctx,
		common.HexToHash(txHash),
	)
	if err != nil {
		return nil, err
	}

	return transaction.GasPrice(), nil
}

func (ec *EthereumClient) TransactionGasUsed(txHash string) (*big.Int, error) {
	ctx := context.TODO()

	receipt, err := ec.client.TransactionReceipt(
		ctx,
		common.HexToHash(txHash),
	)
	if err != nil {
		return nil, err
	}

	return big.NewInt(int64(receipt.GasUsed)), nil
}

func (ec *EthereumClient) TransactionMethod(txHash string) (string, error) {
	ctx := context.TODO()

	transaction, _, err := ec.client.TransactionByHash(
		ctx,
		common.HexToHash(txHash),
	)
	if err != nil {
		return "", err
	}

	for _, lookupAbi := range ec.methodLookupAbiList {
		method, err := lookupAbi.MethodById(transaction.Data()[:4])
		if err != nil {
			continue
		}

		return method.Name, nil
	}

	return "", nil
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

func (ec *EthereumClient) GroupPublicKey(groupIndex int64) ([]byte, error) {
	return ec.operatorContract.GetGroupPublicKey(nil, big.NewInt(groupIndex))
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

func (ec *EthereumClient) GroupMemberRewards(
	groupPublicKey []byte,
) (*big.Int, error) {
	return ec.operatorContract.GetGroupMemberRewards(nil, groupPublicKey)
}

func (ec *EthereumClient) AreRewardsWithdrawn(
	operator string,
	groupIndex int64,
) (bool, error) {
	return ec.operatorContract.HasWithdrawnRewards(
		nil,
		common.HexToAddress(operator),
		big.NewInt(groupIndex),
	)
}

func (ec *EthereumClient) Keeps() (map[int64]string, map[int64]string, error) {
	keepCount, err := ec.keepFactoryContract.GetKeepCount(nil)
	if err != nil {
		return nil, nil, err
	}

	activeKeeps := make(map[int64]string)
	nonActiveKeeps := make(map[int64]string)

	for index := int64(0); index < keepCount.Int64(); index++ {
		address, err := ec.keepFactoryContract.GetKeepAtIndex(
			nil,
			big.NewInt(index),
		)
		if err != nil {
			return nil, nil, err
		}

		keep, err := ec.getKeep(address.Hex())
		if err != nil {
			return nil, nil, err
		}

		isActive, err := keep.IsActive(nil)
		if err != nil {
			return nil, nil, err
		}

		if isActive {
			activeKeeps[index] = address.Hex()
		} else {
			nonActiveKeeps[index] = address.Hex()
		}
	}

	return activeKeeps, nonActiveKeeps, nil
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

func (ec *EthereumClient) KeepMemberBalance(
	keepAddress, memberAddress string,
) (*big.Int, error) {
	keep, err := ec.getKeep(keepAddress)
	if err != nil {
		return nil, err
	}

	return keep.GetMemberETHBalance(nil, common.HexToAddress(memberAddress))
}

func (ec *EthereumClient) getKeep(
	address string,
) (*ecdsaabi.BondedECDSAKeepCaller, error) {
	return ecdsaabi.NewBondedECDSAKeepCaller(
		common.HexToAddress(address),
		ec.client,
	)
}

func WeiToEth(wei *big.Int) *big.Float {
	weiFloat := new(big.Float)
	weiFloat.SetString(wei.String())
	return new(big.Float).Quo(weiFloat, big.NewFloat(math.Pow10(18)))
}

func WeiToGwei(wei *big.Int) *big.Float {
	weiFloat := new(big.Float)
	weiFloat.SetString(wei.String())
	return new(big.Float).Quo(weiFloat, big.NewFloat(math.Pow10(9)))
}
