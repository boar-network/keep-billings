package billing

import (
	"math/big"
	"testing"
)

func TestCalculateBeaconRewards(t *testing.T) {
	tests := map[string]struct {
		customerSharePercentage *big.Float
		beneficiaryEthBalance   *big.Float
		beneficiaryKeepBalance  *big.Float
		accumulatedRewards      *big.Float

		expectedCustomerEthRewardShare  *big.Float
		expectedProviderEthRewardShare  *big.Float
		expectedCustomerKeepRewardShare *big.Float
		expectedProviderKeepRewardShare *big.Float
	}{
		"all non-zero": {
			customerSharePercentage: big.NewFloat(80.0),
			beneficiaryEthBalance:   big.NewFloat(1.22),
			beneficiaryKeepBalance:  big.NewFloat(1.924875),
			accumulatedRewards:      big.NewFloat(0.285758),

			// 0.285758 * 0.8 + 1.22 = 1.4486064
			expectedCustomerEthRewardShare: big.NewFloat(1.4486064),
			// 0.285758 * (1.0 - 0.8) = 0.0571516
			expectedProviderEthRewardShare: big.NewFloat(0.0571516),
			// 1.924875 * 0.8 = 1.5399
			expectedCustomerKeepRewardShare: big.NewFloat(1.5399),
			// 1.924875 * (1.0 - 0.8) = 0.384975
			expectedProviderKeepRewardShare: big.NewFloat(0.384975),
		},
		"zero KEEP rewards": {
			customerSharePercentage: big.NewFloat(70.0),
			beneficiaryEthBalance:   big.NewFloat(4.25),
			beneficiaryKeepBalance:  big.NewFloat(0),
			accumulatedRewards:      big.NewFloat(0.285758),

			// 0.285758 * 0.7 + 4.25 = 4.4500306
			expectedCustomerEthRewardShare: big.NewFloat(4.4500306),
			// 0.285758 * (1.0 - 0.7) = 0.0857274
			expectedProviderEthRewardShare: big.NewFloat(0.0857274),
			// 0 * 0.7 = 0.0
			expectedCustomerKeepRewardShare: big.NewFloat(0),
			// 0 * (1.0 - 0.7) = 0.0
			expectedProviderKeepRewardShare: big.NewFloat(0),
		},
		"zero ETH beneficiary balance": {
			customerSharePercentage: big.NewFloat(70.0),
			beneficiaryEthBalance:   big.NewFloat(0),
			beneficiaryKeepBalance:  big.NewFloat(1.5),
			accumulatedRewards:      big.NewFloat(0.285758),

			// 0.285758 * 0.7 + 0.0 = 0.2000306
			expectedCustomerEthRewardShare: big.NewFloat(0.2000306),
			// 0.285758 * (1.0 - 0.7) = 0.0857274
			expectedProviderEthRewardShare: big.NewFloat(0.0857274),
			// 1.5 * 0.7 = 1.05
			expectedCustomerKeepRewardShare: big.NewFloat(1.05),
			// 1.5 * (1.0 - 0.7) = 0.45
			expectedProviderKeepRewardShare: big.NewFloat(0.45),
		},
		"zero accumulated ETH rewards": {
			customerSharePercentage: big.NewFloat(80.0),
			beneficiaryEthBalance:   big.NewFloat(1.22),
			beneficiaryKeepBalance:  big.NewFloat(1.924875),
			accumulatedRewards:      big.NewFloat(0),

			// 0.0 * 0.8 + 1.22 = 1.22
			expectedCustomerEthRewardShare: big.NewFloat(1.22),
			// 0.0 * (1.0 - 0.8) = 0.0
			expectedProviderEthRewardShare: big.NewFloat(0),
			// 1.924875 * 0.8 = 1.5399
			expectedCustomerKeepRewardShare: big.NewFloat(1.5399),
			// 1.924875 * (1.0 - 0.8) = 0.384975
			expectedProviderKeepRewardShare: big.NewFloat(0.384975),
		},
	}

	for testName, test := range tests {
		t.Run(testName, func(t *testing.T) {
			customerEthRewardsShare, providerEthRewardShare,
				customerKeepRewardShare, providerKeepRewardShare :=
				calculateFinalBeaconRewards(
					test.customerSharePercentage,
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
