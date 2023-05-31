package types_test

import (
	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	ibctesting "github.com/cosmos/ibc-go/v7/testing"
	"github.com/cosmos/ibc-go/v7/testing/mock"
)

func (suite *TypesTestSuite) TestUpgradeValidateBasic() {
	var upgrade types.Upgrade

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"success",
			func() {},
			true,
		},
		{
			"invalid ordering",
			func() {
				upgrade.Fields.Ordering = types.NONE
			},
			false,
		},
		{
			"connection hops length not equal to 1",
			func() {
				upgrade.Fields.ConnectionHops = []string{"connection-0", "connection-1"}
			},
			false,
		},
		{
			"empty version",
			func() {
				upgrade.Fields.Version = ""
			},
			false,
		},
		{
			"invalid timeout",
			func() {
				upgrade.Timeout.Height = clienttypes.ZeroHeight()
				upgrade.Timeout.Timestamp = 0
			},
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			upgrade = types.NewUpgrade(
				types.NewUpgradeFields(types.ORDERED, []string{ibctesting.FirstConnectionID}, mock.Version),
				types.NewTimeout(clienttypes.NewHeight(0, 100), 0),
				0,
			)

			tc.malleate()

			err := upgrade.ValidateBasic()

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}
