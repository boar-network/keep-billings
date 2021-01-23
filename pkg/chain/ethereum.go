package chain

import (
	"context"
	"math"
	"math/big"

	"github.com/ipfs/go-log"

	coreabi "github.com/boar-network/keep-billings/pkg/chain/gen/core/abi"
	ecdsaabi "github.com/boar-network/keep-billings/pkg/chain/gen/ecdsa/abi"
	erc20abi "github.com/boar-network/keep-billings/pkg/chain/gen/erc20/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

var logger = log.Logger("billings-ethereum")

var methodLookupAbiStrings = []string{
	coreabi.TokenStakingABI,
	coreabi.KeepRandomBeaconOperatorABI,
	coreabi.KeepRandomBeaconServiceImplV1ABI,
	ecdsaabi.KeepBondingABI,
}

type EthereumClient struct {
	client           *ethclient.Client
	keepToken        *erc20abi.TokenCaller
	tokenStaking     *coreabi.TokenStakingCaller
	operatorContract *coreabi.KeepRandomBeaconOperatorCaller
}

func NewEthereumClient(
	url string,
	keepTokenAddress string,
	tokenStakingAddress string,
	operatorContractAddress string,
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

	return &EthereumClient{
		client:           client,
		keepToken:        keepToken,
		tokenStaking:     tokenStaking,
		operatorContract: operatorContract,
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
