package billing

import (
	"testing"
	"math/big"
)

func TestCalculateBeaconRewards(t *testing.T) {
	tests := map[string]struct {
		initialOperatorEthBalance *big.Float
		customerSharePercentage *big.Float
		operatorEthBalance *big.Float
		beneficiaryEthBalance *big.Float
		beneficiaryKeepBalance *big.Float
		accumulatedRewards *big.Float

		expectedOperationalCosts *big.Float
		expectedCustomerEthRewardShare *big.Float
		expectedProviderEthRewardShare *big.Float
		expectedCustomerKeepRewardShare *big.Float
		expectedProviderKeepRewardShare *big.Float
	}{
		"negative operational costs (something is wrong)": {
			initialOperatorEthBalance: big.NewFloat(2.0),
			customerSharePercentage: big.NewFloat(50.0),
			operatorEthBalance: big.NewFloat(3.0),
			beneficiaryEthBalance: big.NewFloat(1.0),
			beneficiaryKeepBalance: big.NewFloat(1.0),
			accumulatedRewards: big.NewFloat(1.0),

			expectedOperationalCosts: big.NewFloat(0),
			expectedCustomerEthRewardShare: big.NewFloat(0),
			expectedProviderEthRewardShare: big.NewFloat(0),
			expectedCustomerKeepRewardShare: big.NewFloat(0),
			expectedProviderKeepRewardShare: big.NewFloat(0),
		},
		"no operational costs": {
			initialOperatorEthBalance: big.NewFloat(2.0),
			customerSharePercentage: big.NewFloat(50.0),
			operatorEthBalance: big.NewFloat(2.0),
			beneficiaryEthBalance: big.NewFloat(0.0),
			beneficiaryKeepBalance: big.NewFloat(0.0),
			accumulatedRewards: big.NewFloat(0.0),

			expectedOperationalCosts: big.NewFloat(0),
			expectedCustomerEthRewardShare: big.NewFloat(0),
			expectedProviderEthRewardShare: big.NewFloat(0),
			expectedCustomerKeepRewardShare: big.NewFloat(0),
			expectedProviderKeepRewardShare: big.NewFloat(0),
		},
		"operational cost greater than beneficiary balance": {
			initialOperatorEthBalance: big.NewFloat(2.0),
			customerSharePercentage: big.NewFloat(50.0),
			operatorEthBalance: big.NewFloat(1.0),
			beneficiaryEthBalance: big.NewFloat(0.0),
			beneficiaryKeepBalance: big.NewFloat(0.0),
			accumulatedRewards: big.NewFloat(0.0),

			expectedOperationalCosts: big.NewFloat(0),
			expectedCustomerEthRewardShare: big.NewFloat(0),
			expectedProviderEthRewardShare: big.NewFloat(0),
			expectedCustomerKeepRewardShare: big.NewFloat(0),
			expectedProviderKeepRewardShare: big.NewFloat(0),
		},
		"non-net-zero ETH rewards": {
			initialOperatorEthBalance: big.NewFloat(2.0),
			customerSharePercentage: big.NewFloat(50.0),
			operatorEthBalance: big.NewFloat(1.924875),
			beneficiaryEthBalance: big.NewFloat(0.0),
			beneficiaryKeepBalance: big.NewFloat(0.0),
			accumulatedRewards: big.NewFloat(0.285758),

			// 2.0 - 1.924875 = 0.075125
			expectedOperationalCosts: big.NewFloat(0.075125),
			// 0.285758 - 0.075125 = 0.210633
			// 0.5 * 0.210633 = 0.1053165
			expectedCustomerEthRewardShare: big.NewFloat(0.1053165),
			expectedProviderEthRewardShare: big.NewFloat(0.1053165),
			expectedCustomerKeepRewardShare: big.NewFloat(0),
			expectedProviderKeepRewardShare: big.NewFloat(0),	
		},
		"non-net-zero KEEP rewards": {
			initialOperatorEthBalance: big.NewFloat(2.0),
			customerSharePercentage: big.NewFloat(50.0),
			operatorEthBalance: big.NewFloat(1.0),
			beneficiaryEthBalance: big.NewFloat(0.0),
			beneficiaryKeepBalance: big.NewFloat(9.0),
			accumulatedRewards: big.NewFloat(0.0),

			expectedOperationalCosts: big.NewFloat(0),
			expectedCustomerEthRewardShare: big.NewFloat(0),
			expectedProviderEthRewardShare: big.NewFloat(0),
			expectedCustomerKeepRewardShare: big.NewFloat(4.5),
			expectedProviderKeepRewardShare: big.NewFloat(4.5),
		},
	}

	for testName, test := range tests {
		t.Run(testName, func(t *testing.T) {
			operationalCosts, customerEthRewardsShare,
			providerEthRewardShare, customerKeepRewardShare,
			providerKeepRewardShare := calculateFinalBeaconRewards(
				test.initialOperatorEthBalance,
				test.customerSharePercentage,
				test.operatorEthBalance,
				test.beneficiaryEthBalance,
				test.beneficiaryKeepBalance,
				test.accumulatedRewards,
			)

			assertEqual := func(
				expected *big.Float,
				actual *big.Float,
				description string,
			) {
				float64EqualityThreshold := big.NewFloat(1e-9)
				if new(big.Float).Sub(expected, actual).Cmp(float64EqualityThreshold) > 0 {
					t.Errorf(
						"unexpected %s\nexpected: [%v]\nactual:   [%v]",
						description,
						expected,
						actual,
					)
				}
			}

			assertEqual(
				test.expectedOperationalCosts,
				operationalCosts,
				"operational costs",
			)
			assertEqual(
				test.expectedCustomerEthRewardShare,
				customerEthRewardsShare,
				"customer ETH reward share",
			)
			assertEqual(
				test.expectedProviderEthRewardShare,
				providerEthRewardShare,
				"provider ETH reward share",
			)
			assertEqual(
				test.expectedCustomerKeepRewardShare,
				customerKeepRewardShare,
				"customer KEEP reward share",
			)
			assertEqual(
				test.expectedProviderKeepRewardShare,
				providerKeepRewardShare,
				"provider KEEP reward share",
			)
		})
	}
}