package keeper_test

import (
	"fmt"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	connectiontypes "github.com/cosmos/ibc-go/v7/modules/core/03-connection/types"
	"github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	commitmenttypes "github.com/cosmos/ibc-go/v7/modules/core/23-commitment/types"
	host "github.com/cosmos/ibc-go/v7/modules/core/24-host"
	"github.com/cosmos/ibc-go/v7/modules/core/exported"
	ibctesting "github.com/cosmos/ibc-go/v7/testing"
	"github.com/cosmos/ibc-go/v7/testing/mock"
)

func (suite *KeeperTestSuite) TestChanUpgradeInit() {
	var (
		path          *ibctesting.Path
		expSequence   uint64
		upgradeFields types.UpgradeFields
	)

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
			"success with later upgrade sequence",
			func() {
				channel := path.EndpointA.GetChannel()
				channel.UpgradeSequence = 4
				path.EndpointA.SetChannel(channel)
				expSequence = 5
			},
			true,
		},
		{
			"upgrade fields are identical to channel end",
			func() {
				channel := path.EndpointA.GetChannel()
				upgradeFields = types.NewUpgradeFields(channel.Ordering, channel.ConnectionHops, channel.Version)
			},
			false,
		},
		{
			"channel not found",
			func() {
				path.EndpointA.ChannelID = "invalid-channel"
				path.EndpointA.ChannelConfig.PortID = "invalid-port"
			},
			false,
		},
		{
			"channel state is not in OPEN state",
			func() {
				suite.Require().NoError(path.EndpointA.SetChannelState(types.CLOSED))
			},
			false,
		},
		{
			"proposed channel connection not found",
			func() {
				upgradeFields.ConnectionHops = []string{"connection-100"}
			},
			false,
		},
		{
			"invalid proposed channel connection state",
			func() {
				connectionEnd := path.EndpointA.GetConnection()
				connectionEnd.State = connectiontypes.UNINITIALIZED

				suite.chainA.GetSimApp().GetIBCKeeper().ConnectionKeeper.SetConnection(suite.chainA.GetContext(), "connection-100", connectionEnd)
				upgradeFields.ConnectionHops = []string{"connection-100"}
			},
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTest()

			path = ibctesting.NewPath(suite.chainA, suite.chainB)
			suite.coordinator.Setup(path)

			expSequence = 1

			upgradeFields = types.NewUpgradeFields(types.UNORDERED, []string{path.EndpointA.ConnectionID}, mock.UpgradeVersion)

			tc.malleate()

			upgrade, err := suite.chainA.GetSimApp().IBCKeeper.ChannelKeeper.ChanUpgradeInit(
				suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, upgradeFields,
			)

			if tc.expPass {
				ctx := suite.chainA.GetContext()
				suite.chainA.GetSimApp().IBCKeeper.ChannelKeeper.WriteUpgradeInitChannel(ctx, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, upgrade)
				channel := path.EndpointA.GetChannel()

				events := ctx.EventManager().Events().ToABCIEvents()
				expEvents := ibctesting.EventsMap{
					types.EventTypeChannelUpgradeInit: {
						types.AttributeKeyPortID:                    path.EndpointA.ChannelConfig.PortID,
						types.AttributeKeyChannelID:                 path.EndpointA.ChannelID,
						types.AttributeCounterpartyPortID:           path.EndpointB.ChannelConfig.PortID,
						types.AttributeCounterpartyChannelID:        path.EndpointB.ChannelID,
						types.AttributeKeyUpgradeConnectionHops:     upgradeFields.ConnectionHops[0],
						types.AttributeKeyUpgradeVersion:            upgradeFields.Version,
						types.AttributeKeyUpgradeOrdering:           upgradeFields.Ordering.String(),
						types.AttributeKeyUpgradeSequence:           fmt.Sprintf("%d", channel.UpgradeSequence),
						types.AttributeKeyUpgradeChannelFlushStatus: channel.FlushStatus.String(),
					},
					sdk.EventTypeMessage: {
						sdk.AttributeKeyModule: types.AttributeValueCategory,
					},
				}

				suite.Require().NoError(err)
				suite.Require().Equal(expSequence, channel.UpgradeSequence)
				suite.Require().Equal(mock.Version, channel.Version)
				suite.Require().Equal(types.OPEN, channel.State)
				ibctesting.AssertEvents(&suite.Suite, expEvents, events)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestChanUpgradeTry() {
	var (
		path                *ibctesting.Path
		proposedUpgrade     types.Upgrade
		counterpartyUpgrade types.Upgrade
	)

	testCases := []struct {
		name     string
		malleate func()
		expError error
	}{
		{
			"success",
			func() {},
			nil,
		},
		{
			"success: crossing hellos",
			func() {
				err := path.EndpointB.ChanUpgradeInit()
				suite.Require().NoError(err)
			},
			nil,
		},
		// {
		// 	"success: upgrade sequence is fast forwarded to counterparty upgrade sequence",
		// 	func() {
		// 		channel := path.EndpointA.GetChannel()
		// 		channel.UpgradeSequence = 5
		// 		path.EndpointA.SetChannel(channel)

		// 		expSequence = 5
		// 	},
		// 	true,
		// },
		// {
		{
			"channel not found",
			func() {
				path.EndpointB.ChannelID = ibctesting.InvalidID
			},
			types.ErrChannelNotFound,
		},
		{
			"channel state is not in OPEN or INITUPGRADE state",
			func() {
				suite.Require().NoError(path.EndpointB.SetChannelState(types.CLOSED))
			},
			types.ErrInvalidChannelState,
		},
		{
			"connection not found",
			func() {
				channel := path.EndpointB.GetChannel()
				channel.ConnectionHops = []string{"connection-100"}
				path.EndpointB.SetChannel(channel)
			},
			connectiontypes.ErrConnectionNotFound,
		},
		{
			"invalid connection state",
			func() {
				connectionEnd := path.EndpointB.GetConnection()
				connectionEnd.State = connectiontypes.UNINITIALIZED
				suite.chainB.GetSimApp().GetIBCKeeper().ConnectionKeeper.SetConnection(suite.chainB.GetContext(), path.EndpointB.ConnectionID, connectionEnd)
			},
			connectiontypes.ErrInvalidConnectionState,
		},
		{
			"initializing handshake fails, proposed connection hops do not exist",
			func() {
				proposedUpgrade.Fields.ConnectionHops = []string{ibctesting.InvalidID}
			},
			connectiontypes.ErrConnectionNotFound,
		},
		{
			"fails due to proof verification failure, counterparty channel ordering does not match expected ordering",
			func() {
				channel := path.EndpointB.GetChannel()
				channel.Ordering = types.ORDERED
				path.EndpointB.SetChannel(channel)
			},
			commitmenttypes.ErrInvalidProof,
		},
		{
			"fails due to proof verification failure, counterparty upgrade connection hops are tampered with",
			func() {
				counterpartyUpgrade.Fields.ConnectionHops = []string{ibctesting.InvalidID}
			},
			types.ErrIncompatibleCounterpartyUpgrade,
		},
		{
			"fails due to incompatible upgrades, chainB proposes a new connection hop that does not match counterparty",
			func() {
				// reuse existing connection to create a new connection in a non OPEN state
				connection := path.EndpointB.GetConnection()
				// ensure counterparty connectionID does not match connectionID set in counterparty proposed upgrade
				connection.Counterparty.ConnectionId = "connection-50"

				// set proposed connection in state
				proposedConnectionID := "connection-100" //nolint:goconst
				suite.chainB.GetSimApp().GetIBCKeeper().ConnectionKeeper.SetConnection(suite.chainB.GetContext(), proposedConnectionID, connection)
				proposedUpgrade.Fields.ConnectionHops[0] = proposedConnectionID
			},
			types.ErrIncompatibleCounterpartyUpgrade,
		},
		{
			"fails due to mismatch in upgrade sequences",
			func() {
				channel := path.EndpointB.GetChannel()
				channel.UpgradeSequence = 5
				path.EndpointB.SetChannel(channel)
			},
			types.NewUpgradeError(5, types.ErrInvalidUpgradeSequence), // channel sequence - 1 will be returned
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTest()
			expPass := tc.expError == nil

			path = ibctesting.NewPath(suite.chainA, suite.chainB)
			suite.coordinator.Setup(path)

			path.EndpointA.ChannelConfig.ProposedUpgrade.Fields.Version = mock.UpgradeVersion
			err := path.EndpointA.ChanUpgradeInit()
			suite.Require().NoError(err)

			path.EndpointB.ChannelConfig.ProposedUpgrade.Fields.Version = mock.UpgradeVersion
			proposedUpgrade = path.EndpointB.GetProposedUpgrade()

			var found bool
			counterpartyUpgrade, found = path.EndpointA.Chain.GetSimApp().IBCKeeper.ChannelKeeper.GetUpgrade(path.EndpointA.Chain.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
			suite.Require().True(found)

			tc.malleate()

			// ensure clients are up to date to receive valid proofs
			suite.Require().NoError(path.EndpointB.UpdateClient())

			proofCounterpartyChannel, proofCounterpartyUpgrade, proofHeight := path.EndpointB.QueryChannelUpgradeProof()

			upgrade, err := suite.chainB.GetSimApp().IBCKeeper.ChannelKeeper.ChanUpgradeTry(
				suite.chainB.GetContext(),
				path.EndpointB.ChannelConfig.PortID,
				path.EndpointB.ChannelID,
				proposedUpgrade.Fields.ConnectionHops,
				counterpartyUpgrade.Fields,
				path.EndpointA.GetChannel().UpgradeSequence,
				proofCounterpartyChannel,
				proofCounterpartyUpgrade,
				proofHeight,
			)

			if expPass {
				suite.Require().NoError(err)
				suite.Require().NotEmpty(upgrade)
				suite.Require().Equal(proposedUpgrade.Fields, upgrade.Fields)

				latestSequenceSend, found := path.EndpointB.Chain.GetSimApp().IBCKeeper.ChannelKeeper.GetNextSequenceSend(path.EndpointB.Chain.GetContext(), path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
				suite.Require().True(found)
				suite.Require().Equal(latestSequenceSend-1, upgrade.LatestSequenceSend)
			} else {
				suite.assertUpgradeError(err, tc.expError)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestWriteUpgradeTry() {
	var (
		path            *ibctesting.Path
		proposedUpgrade types.Upgrade
	)

	testCases := []struct {
		name                 string
		malleate             func()
		hasPacketCommitments bool
	}{
		{
			"success with no packet commitments",
			func() {},
			false,
		},
		{
			"success with packet commitments",
			func() {
				// manually set packet commitment
				sequence, err := path.EndpointB.SendPacket(suite.chainB.GetTimeoutHeight(), 0, ibctesting.MockPacketData)
				suite.Require().NoError(err)
				suite.Require().Equal(uint64(1), sequence)
			},
			true,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTest()

			path = ibctesting.NewPath(suite.chainA, suite.chainB)
			suite.coordinator.Setup(path)

			path.EndpointB.ChannelConfig.ProposedUpgrade.Fields.Version = mock.UpgradeVersion
			proposedUpgrade = path.EndpointB.GetProposedUpgrade()

			tc.malleate()

			ctx := suite.chainB.GetContext()
			upgradedChannelEnd, upgradeWithAppCallbackVersion := suite.chainB.GetSimApp().IBCKeeper.ChannelKeeper.WriteUpgradeTryChannel(
				ctx,
				path.EndpointB.ChannelConfig.PortID,
				path.EndpointB.ChannelID,
				proposedUpgrade,
				proposedUpgrade.Fields.Version,
				proposedUpgrade.LatestSequenceSend,
			)

			channel := path.EndpointB.GetChannel()
			suite.Require().Equal(upgradedChannelEnd, channel)

			upgrade, found := suite.chainB.GetSimApp().IBCKeeper.ChannelKeeper.GetUpgrade(suite.chainB.GetContext(), path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
			suite.Require().True(found)
			suite.Require().Equal(types.TRYUPGRADE, channel.State)
			suite.Require().Equal(upgradeWithAppCallbackVersion, upgrade)

			actualCounterpartyLastSequenceSend, ok := suite.chainB.GetSimApp().IBCKeeper.ChannelKeeper.GetCounterpartyLastPacketSequence(suite.chainB.GetContext(), path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
			suite.Require().True(ok)
			suite.Require().Equal(proposedUpgrade.LatestSequenceSend, actualCounterpartyLastSequenceSend)

			events := ctx.EventManager().Events().ToABCIEvents()
			expEvents := ibctesting.EventsMap{
				types.EventTypeChannelUpgradeTry: {
					types.AttributeKeyPortID:                    path.EndpointB.ChannelConfig.PortID,
					types.AttributeKeyChannelID:                 path.EndpointB.ChannelID,
					types.AttributeCounterpartyPortID:           path.EndpointA.ChannelConfig.PortID,
					types.AttributeCounterpartyChannelID:        path.EndpointA.ChannelID,
					types.AttributeKeyUpgradeConnectionHops:     upgrade.Fields.ConnectionHops[0],
					types.AttributeKeyUpgradeVersion:            upgrade.Fields.Version,
					types.AttributeKeyUpgradeOrdering:           upgrade.Fields.Ordering.String(),
					types.AttributeKeyUpgradeSequence:           fmt.Sprintf("%d", channel.UpgradeSequence),
					types.AttributeKeyUpgradeChannelFlushStatus: channel.FlushStatus.String(),
				},
				sdk.EventTypeMessage: {
					sdk.AttributeKeyModule: types.AttributeValueCategory,
				},
			}

			ibctesting.AssertEvents(&suite.Suite, expEvents, events)

			if tc.hasPacketCommitments {
				suite.Require().Equal(types.FLUSHING, channel.FlushStatus)
			} else {
				suite.Require().Equal(types.FLUSHCOMPLETE, channel.FlushStatus)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestChanUpgradeAck() {
	var (
		path                *ibctesting.Path
		counterpartyUpgrade types.Upgrade
	)

	testCases := []struct {
		name     string
		malleate func()
		expError error
	}{
		// TODO: uncomment and handle failing tests
		// {
		// 	"success",
		// 	func() {},
		// 	nil,
		// },
		// {
		// 	"success with later upgrade sequence",
		// 	func() {
		// 		channel := path.EndpointA.GetChannel()
		// 		channel.UpgradeSequence = 10
		// 		path.EndpointA.SetChannel(channel)

		// 		channel = path.EndpointB.GetChannel()
		// 		channel.UpgradeSequence = 10
		// 		path.EndpointB.SetChannel(channel)

		// 		suite.coordinator.CommitBlock(suite.chainA, suite.chainB)

		// 		err := path.EndpointA.UpdateClient()
		// 		suite.Require().NoError(err)
		// 	},
		// 	nil,
		// },
		{
			"channel not found",
			func() {
				path.EndpointA.ChannelID = ibctesting.InvalidID
				path.EndpointA.ChannelConfig.PortID = ibctesting.InvalidID
			},
			types.ErrChannelNotFound,
		},
		{
			"channel state is not in INITUPGRADE or TRYUPGRADE state",
			func() {
				suite.Require().NoError(path.EndpointA.SetChannelState(types.CLOSED))
			},
			types.ErrInvalidChannelState,
		},
		{
			"connection not found",
			func() {
				channel := path.EndpointA.GetChannel()
				channel.ConnectionHops = []string{"connection-100"}
				path.EndpointA.SetChannel(channel)
			},
			connectiontypes.ErrConnectionNotFound,
		},
		{
			"invalid connection state",
			func() {
				connectionEnd := path.EndpointA.GetConnection()
				connectionEnd.State = connectiontypes.UNINITIALIZED
				path.EndpointA.SetConnection(connectionEnd)
			},
			connectiontypes.ErrInvalidConnectionState,
		},
		{
			"upgrade not found",
			func() {
				store := suite.chainA.GetContext().KVStore(suite.chainA.GetSimApp().GetKey(exported.ModuleName))
				store.Delete(host.ChannelUpgradeKey(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID))
			},
			types.ErrUpgradeNotFound,
		},
		{
			"fails due to upgrade incompatibility",
			func() {
				// Need to set counterparty upgrade in state and update clients to ensure
				// proofs submitted reflect the altered upgrade.
				counterpartyUpgrade.Fields.ConnectionHops = []string{ibctesting.InvalidID}
				path.EndpointB.SetChannelUpgrade(counterpartyUpgrade)

				suite.coordinator.CommitBlock(suite.chainB)

				err := path.EndpointA.UpdateClient()
				suite.Require().NoError(err)
			},
			types.NewUpgradeError(1, types.ErrIncompatibleCounterpartyUpgrade),
		},
		{
			"fails due to proof verification failure, counterparty channel ordering does not match expected ordering",
			func() {
				channel := path.EndpointA.GetChannel()
				channel.Ordering = types.ORDERED
				path.EndpointA.SetChannel(channel)
			},
			commitmenttypes.ErrInvalidProof,
		},
		{
			"fails due to proof verification failure, counterparty update has unexpected sequence",
			func() {
				// Decrementing LatestSequenceSend is sufficient to cause the proof to fail.
				counterpartyUpgrade.LatestSequenceSend--
			},
			commitmenttypes.ErrInvalidProof,
		},
		{
			"fails due to mismatch in upgrade ordering",
			func() {
				upgrade := path.EndpointA.GetChannelUpgrade()
				upgrade.Fields.Ordering = types.NONE

				path.EndpointA.SetChannelUpgrade(upgrade)
			},
			types.NewUpgradeError(1, types.ErrIncompatibleCounterpartyUpgrade),
		},
		// {
		// 	"channel end version mismatch on crossing hellos",
		// 	func() {
		// 		channel := path.EndpointA.GetChannel()
		// 		channel.State = types.TRYUPGRADE

		// 		path.EndpointA.SetChannel(channel)

		// 		upgrade := path.EndpointA.GetChannelUpgrade()
		// 		upgrade.Fields.Version = "invalid-version"

		// 		path.EndpointA.SetChannelUpgrade(upgrade)
		// 	},
		// 	types.NewUpgradeError(1, types.ErrIncompatibleCounterpartyUpgrade),
		// },
		{
			"counterparty timeout has elapsed",
			func() {
				// Need to set counterparty upgrade in state and update clients to ensure
				// proofs submitted reflect the altered upgrade.
				counterpartyUpgrade.Timeout = types.NewTimeout(clienttypes.NewHeight(0, 1), 0)
				path.EndpointB.SetChannelUpgrade(counterpartyUpgrade)

				err := path.EndpointB.UpdateClient()
				suite.Require().NoError(err)
				err = path.EndpointA.UpdateClient()
				suite.Require().NoError(err)
			},
			types.NewUpgradeError(1, types.ErrInvalidUpgrade),
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTest()

			path = ibctesting.NewPath(suite.chainA, suite.chainB)
			suite.coordinator.Setup(path)

			path.EndpointA.ChannelConfig.ProposedUpgrade.Fields.Version = mock.UpgradeVersion
			path.EndpointB.ChannelConfig.ProposedUpgrade.Fields.Version = mock.UpgradeVersion

			err := path.EndpointA.ChanUpgradeInit()
			suite.Require().NoError(err)

			// manually set packet commitment so that the chainB channel flush status is FLUSHING
			sequence, err := path.EndpointB.SendPacket(suite.chainB.GetTimeoutHeight(), 0, ibctesting.MockPacketData)
			suite.Require().NoError(err)
			suite.Require().Equal(uint64(1), sequence)

			err = path.EndpointB.ChanUpgradeTry()
			suite.Require().NoError(err)

			// ensure client is up to date to receive valid proofs
			err = path.EndpointA.UpdateClient()
			suite.Require().NoError(err)

			counterpartyUpgrade = path.EndpointB.GetChannelUpgrade()

			tc.malleate()

			proofChannel, proofUpgrade, proofHeight := path.EndpointA.QueryChannelUpgradeProof()

			err = suite.chainA.GetSimApp().IBCKeeper.ChannelKeeper.ChanUpgradeAck(
				suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, counterpartyUpgrade,
				proofChannel, proofUpgrade, proofHeight,
			)

			expPass := tc.expError == nil
			if expPass {
				suite.Require().NoError(err)
			} else {
				suite.assertUpgradeError(err, tc.expError)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestWriteChannelUpgradeAck() {
	var (
		path            *ibctesting.Path
		proposedUpgrade types.Upgrade
	)

	testCases := []struct {
		name                 string
		malleate             func()
		hasPacketCommitments bool
	}{
		{
			"success with no packet commitments",
			func() {},
			false,
		},
		{
			"success with packet commitments",
			func() {
				// manually set packet commitment
				sequence, err := path.EndpointA.SendPacket(suite.chainB.GetTimeoutHeight(), 0, ibctesting.MockPacketData)
				suite.Require().NoError(err)
				suite.Require().Equal(uint64(1), sequence)
			},
			true,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTest()

			path = ibctesting.NewPath(suite.chainA, suite.chainB)
			suite.coordinator.Setup(path)

			path.EndpointA.ChannelConfig.ProposedUpgrade.Fields.Version = mock.UpgradeVersion
			path.EndpointB.ChannelConfig.ProposedUpgrade.Fields.Version = mock.UpgradeVersion

			tc.malleate()

			// perform the upgrade handshake.
			suite.Require().NoError(path.EndpointA.ChanUpgradeInit())

			suite.Require().NoError(path.EndpointB.ChanUpgradeTry())

			ctx := suite.chainA.GetContext()
			proposedUpgrade = path.EndpointB.GetChannelUpgrade()

			suite.chainA.GetSimApp().IBCKeeper.ChannelKeeper.WriteUpgradeAckChannel(ctx, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, proposedUpgrade)

			channel := path.EndpointA.GetChannel()
			upgrade := path.EndpointA.GetChannelUpgrade()
			suite.Require().Equal(mock.UpgradeVersion, upgrade.Fields.Version)

			actualCounterpartyLastSequenceSend, ok := suite.chainA.GetSimApp().IBCKeeper.ChannelKeeper.GetCounterpartyLastPacketSequence(suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
			suite.Require().True(ok)
			suite.Require().Equal(proposedUpgrade.LatestSequenceSend, actualCounterpartyLastSequenceSend)

			events := ctx.EventManager().Events().ToABCIEvents()
			expEvents := ibctesting.EventsMap{
				types.EventTypeChannelUpgradeAck: {
					types.AttributeKeyPortID:                    path.EndpointA.ChannelConfig.PortID,
					types.AttributeKeyChannelID:                 path.EndpointA.ChannelID,
					types.AttributeCounterpartyPortID:           path.EndpointB.ChannelConfig.PortID,
					types.AttributeCounterpartyChannelID:        path.EndpointB.ChannelID,
					types.AttributeKeyUpgradeConnectionHops:     upgrade.Fields.ConnectionHops[0],
					types.AttributeKeyUpgradeVersion:            upgrade.Fields.Version,
					types.AttributeKeyUpgradeOrdering:           upgrade.Fields.Ordering.String(),
					types.AttributeKeyUpgradeSequence:           fmt.Sprintf("%d", channel.UpgradeSequence),
					types.AttributeKeyUpgradeChannelFlushStatus: channel.FlushStatus.String(),
				},
				sdk.EventTypeMessage: {
					sdk.AttributeKeyModule: types.AttributeValueCategory,
				},
			}

			ibctesting.AssertEvents(&suite.Suite, expEvents, events)

			counterpartyUpgrade, ok := suite.chainA.GetSimApp().IBCKeeper.ChannelKeeper.GetCounterpartyUpgrade(suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
			suite.Require().True(ok)
			suite.Require().Equal(proposedUpgrade, counterpartyUpgrade)

			if tc.hasPacketCommitments {
				suite.Require().Equal(types.FLUSHING, channel.FlushStatus)
			} else {
				suite.Require().Equal(types.FLUSHCOMPLETE, channel.FlushStatus)
			}
		})
	}
}

// TODO: Uncomment and address testcases when appropriate, timeout logic currently causes failures
// func (suite *KeeperTestSuite) TestChanUpgradeOpen() {
// 	var path *ibctesting.Path
// 	testCases := []struct {
// 		name     string
// 		malleate func()
// 		expError error
// 	}{
// 		{
// 			"success",
// 			func() {},
// 			nil,
// 		},
// 		{
// 			"channel not found",
// 			func() {
// 				path.EndpointA.ChannelConfig.PortID = ibctesting.InvalidID
// 			},
// 			types.ErrChannelNotFound,
// 		},

// 		{
// 			"channel state is not in TRYUPGRADE or ACKUPGRADE",
// 			func() {
// 				suite.Require().NoError(path.EndpointA.SetChannelState(types.OPEN))
// 			},
// 			types.ErrInvalidChannelState,
// 		},

// 		{
// 			"channel has in-flight packets",
// 			func() {
// 				portID := path.EndpointA.ChannelConfig.PortID
// 				channelID := path.EndpointA.ChannelID
// 				// Set a dummy packet commitment to simulate in-flight packets
// 				suite.chainA.GetSimApp().IBCKeeper.ChannelKeeper.SetPacketCommitment(suite.chainA.GetContext(), portID, channelID, 1, []byte("hash"))
// 			},
// 			types.ErrPendingInflightPackets,
// 		},
// 		{
// 			"flush status is FLUSHING",
// 			func() {
// 				channel := path.EndpointA.GetChannel()
// 				channel.FlushStatus = types.FLUSHING
// 				path.EndpointA.SetChannel(channel)
// 			},
// 			types.ErrInvalidFlushStatus,
// 		},
// 		{
// 			"flush status is NOTINFLUSH",
// 			func() {
// 				channel := path.EndpointA.GetChannel()
// 				channel.FlushStatus = types.NOTINFLUSH
// 				path.EndpointA.SetChannel(channel)
// 			},
// 			types.ErrInvalidFlushStatus,
// 		},
// 		{
// 			"connection not found",
// 			func() {
// 				channel := path.EndpointA.GetChannel()
// 				channel.ConnectionHops = []string{"connection-100"}
// 				path.EndpointA.SetChannel(channel)
// 			},
// 			connectiontypes.ErrConnectionNotFound,
// 		},
// 		{
// 			"invalid connection state",
// 			func() {
// 				connectionEnd := path.EndpointA.GetConnection()
// 				connectionEnd.State = connectiontypes.UNINITIALIZED
// 				path.EndpointA.SetConnection(connectionEnd)
// 			},
// 			connectiontypes.ErrInvalidConnectionState,
// 		},
// 	}

// 	// Create an initial path used only to invoke a ChanOpenInit handshake.
// 	// This bumps the channel identifier generated for chain A on the
// 	// next path used to run the upgrade handshake.
// 	// See issue 4062.
// 	path = ibctesting.NewPath(suite.chainA, suite.chainB)
// 	suite.coordinator.SetupConnections(path)
// 	suite.Require().NoError(path.EndpointA.ChanOpenInit())

// 	for _, tc := range testCases {
// 		tc := tc
// 		suite.Run(tc.name, func() {
// 			suite.SetupTest()

// 			path = ibctesting.NewPath(suite.chainA, suite.chainB)
// 			suite.coordinator.Setup(path)

// 			path.EndpointB.ChannelConfig.ProposedUpgrade.Fields.Version = mock.UpgradeVersion

// 			err := path.EndpointB.ChanUpgradeInit()
// 			suite.Require().NoError(err)

// 			err = path.EndpointA.ChanUpgradeTry()
// 			suite.Require().NoError(err)

// 			err = path.EndpointB.ChanUpgradeAck()
// 			suite.Require().NoError(err)

// 			suite.coordinator.CommitBlock(suite.chainA, suite.chainB)
// 			suite.Require().NoError(path.EndpointA.UpdateClient())

// 			tc.malleate()

// 			proofCounterpartyChannel, _, proofHeight := path.EndpointA.QueryChannelUpgradeProof()
// 			err = suite.chainA.GetSimApp().IBCKeeper.ChannelKeeper.ChanUpgradeOpen(
// 				suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID,
// 				path.EndpointB.GetChannel().State, proofCounterpartyChannel, proofHeight,
// 			)

// 			if tc.expError == nil {
// 				suite.Require().NoError(err)
// 			} else {
// 				suite.Require().ErrorIs(err, tc.expError)
// 			}
// 		})
// 	}
// }

// TODO: Uncomment and address testcases when appropriate, timeout logic currently causes failures
// TestChanUpgradeOpenCounterPartyStates tests the handshake in the cases where
// the counterparty is in a state other than OPEN.
// func (suite *KeeperTestSuite) TestChanUpgradeOpenCounterpartyStates() {
// 	var path *ibctesting.Path
// 	testCases := []struct {
// 		name     string
// 		malleate func()
// 		expError error
// 	}{
// 		{
// 			"success, counterparty in OPEN",
// 			func() {
// 				err := path.EndpointB.ChanUpgradeInit()
// 				suite.Require().NoError(err)

// 				err = path.EndpointA.ChanUpgradeTry()
// 				suite.Require().NoError(err)

// 				err = path.EndpointB.ChanUpgradeAck()
// 				suite.Require().NoError(err)

// 				suite.coordinator.CommitBlock(suite.chainA, suite.chainB)
// 				suite.Require().NoError(path.EndpointA.UpdateClient())
// 			},
// 			nil,
// 		},
// 		{
// 			"success, counterparty in TRYUPGRADE",
// 			func() {
// 				// Need to create a packet commitment on A so as to keep it from going to OPEN if no inflight packets exist.
// 				sequence, err := path.EndpointA.SendPacket(defaultTimeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
// 				suite.Require().NoError(err)
// 				packet := types.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
// 				err = path.EndpointB.RecvPacket(packet)
// 				suite.Require().NoError(err)

// 				err = path.EndpointA.ChanUpgradeInit()
// 				suite.Require().NoError(err)

// 				err = path.EndpointB.ChanUpgradeTry()
// 				suite.Require().NoError(err)

// 				err = path.EndpointA.ChanUpgradeAck()
// 				suite.Require().NoError(err)

// 				// Ack packet to delete packet commitment before calling ChanUpgradeOpen
// 				err = path.EndpointA.AcknowledgePacket(packet, ibctesting.MockAcknowledgement)
// 				suite.Require().NoError(err)
// 			},
// 			nil,
// 		},
// 	}

// 	// Create an initial path used only to invoke ConnOpenInit/ChanOpenInit handlers.
// 	// This bumps the connection/channel identifiers generated for chain A on the
// 	// next path used to run the upgrade handshake.
// 	// See issue 4062.
// 	path = ibctesting.NewPath(suite.chainA, suite.chainB)
// 	suite.coordinator.SetupClients(path)
// 	suite.Require().NoError(path.EndpointA.ConnOpenInit())
// 	suite.coordinator.SetupConnections(path)
// 	suite.Require().NoError(path.EndpointA.ChanOpenInit())

// 	for _, tc := range testCases {
// 		tc := tc
// 		suite.Run(tc.name, func() {
// 			suite.SetupTest()

// 			path = ibctesting.NewPath(suite.chainA, suite.chainB)
// 			suite.coordinator.Setup(path)

// 			path.EndpointB.ChannelConfig.ProposedUpgrade.Fields.Version = mock.UpgradeVersion
// 			path.EndpointA.ChannelConfig.ProposedUpgrade.Fields.Version = mock.UpgradeVersion

// 			tc.malleate()

// 			proofCounterpartyChannel, _, proofHeight := path.EndpointA.QueryChannelUpgradeProof()
// 			err := suite.chainA.GetSimApp().IBCKeeper.ChannelKeeper.ChanUpgradeOpen(
// 				suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID,
// 				path.EndpointB.GetChannel().State, proofCounterpartyChannel, proofHeight,
// 			)

// 			expPass := tc.expError == nil
// 			if expPass {
// 				suite.Require().NoError(err)
// 			} else {
// 				suite.Require().ErrorIs(err, tc.expError)
// 			}
// 		})
// 	}
// }

// TODO: Uncomment and address testcases when appropriate, timeout logic currently causes failures
// func (suite *KeeperTestSuite) TestWriteUpgradeOpenChannel() {
// 	suite.SetupTest()

// 	path := ibctesting.NewPath(suite.chainA, suite.chainB)
// 	suite.coordinator.Setup(path)

// 	// Need to create a packet commitment on A so as to keep it from going to OPEN if no inflight packets exist.
// 	sequence, err := path.EndpointA.SendPacket(defaultTimeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
// 	suite.Require().NoError(err)
// 	packet := types.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
// 	err = path.EndpointB.RecvPacket(packet)
// 	suite.Require().NoError(err)

// 	path.EndpointA.ChannelConfig.ProposedUpgrade.Fields.Version = mock.UpgradeVersion
// 	path.EndpointB.ChannelConfig.ProposedUpgrade.Fields.Version = mock.UpgradeVersion
// 	path.EndpointA.ChannelConfig.ProposedUpgrade.Fields.Ordering = types.ORDERED
// 	path.EndpointB.ChannelConfig.ProposedUpgrade.Fields.Ordering = types.ORDERED

// 	suite.Require().NoError(path.EndpointA.ChanUpgradeInit())
// 	suite.Require().NoError(path.EndpointB.ChanUpgradeTry())
// 	suite.Require().NoError(path.EndpointA.ChanUpgradeAck())

// 	// Ack packet to delete packet commitment before calling WriteUpgradeOpenChannel
// 	err = path.EndpointA.AcknowledgePacket(packet, ibctesting.MockAcknowledgement)
// 	suite.Require().NoError(err)

//      ctx := suite.chainA.GetContext()
// 	suite.chainA.GetSimApp().IBCKeeper.ChannelKeeper.WriteUpgradeOpenChannel(ctx, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
// 	channel := path.EndpointA.GetChannel()

// 	// Assert that channel state has been updated
// 	suite.Require().Equal(types.OPEN, channel.State)
// 	suite.Require().Equal(mock.UpgradeVersion, channel.Version)
// 	suite.Require().Equal(types.ORDERED, channel.Ordering)
// 	suite.Require().Equal(types.NOTINFLUSH, channel.FlushStatus)

// 	// Assert that state stored for upgrade has been deleted
// 	upgrade, found := suite.chainA.GetSimApp().IBCKeeper.ChannelKeeper.GetUpgrade(suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
// 	suite.Require().Equal(types.Upgrade{}, upgrade)
// 	suite.Require().False(found)

// 	lastPacketSequence, found := suite.chainA.GetSimApp().IBCKeeper.ChannelKeeper.GetCounterpartyLastPacketSequence(suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
// 	suite.Require().Equal(uint64(0), lastPacketSequence)
// 	suite.Require().False(found)

//      events := ctx.EventManager().Events().ToABCIEvents()
//	expEvents := ibctesting.EventsMap{
//		types.EventTypeChannelUpgradeOpen: {
//			types.AttributeKeyPortID:                    path.EndpointA.ChannelConfig.PortID,
//			types.AttributeKeyChannelID:                 path.EndpointA.ChannelID,
//			types.AttributeCounterpartyPortID:           path.EndpointB.ChannelConfig.PortID,
//			types.AttributeCounterpartyChannelID:        path.EndpointB.ChannelID,
//			types.AttributeKeyChannelState:              types.OPEN.String(),
//			types.AttributeKeyUpgradeConnectionHops:     channel.ConnectionHops[0],
//			types.AttributeKeyUpgradeVersion:            channel.Version,
//			types.AttributeKeyUpgradeOrdering:           channel.Ordering.String(),
//			types.AttributeKeyUpgradeSequence:           fmt.Sprintf("%d", channel.UpgradeSequence),
//			types.AttributeKeyUpgradeChannelFlushStatus: channel.FlushStatus.String(),
//		},
//		sdk.EventTypeMessage: {
//			sdk.AttributeKeyModule: types.AttributeValueCategory,
//		},
//	}
//	ibctesting.AssertEvents(&suite.Suite, expEvents, events)

// 	counterpartyUpgrade, found := suite.chainA.GetSimApp().IBCKeeper.ChannelKeeper.GetCounterpartyUpgrade(suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
// 	suite.Require().Equal(types.Upgrade{}, counterpartyUpgrade)
// 	suite.Require().False(found)
// }

// func (suite *KeeperTestSuite) TestChanUpgradeCancel() {
// 	var (
// 		path              *ibctesting.Path
// 		errorReceipt      types.ErrorReceipt
// 		errorReceiptProof []byte
// 		proofHeight       clienttypes.Height
// 	)

// 	tests := []struct {
// 		name     string
// 		malleate func()
// 		expError error
// 	}{
// 		{
// 			name:     "success",
// 			malleate: func() {},
// 			expError: nil,
// 		},
// 		{
// 			name: "invalid channel state",
// 			malleate: func() {
// 				channel := path.EndpointA.GetChannel()
// 				channel.State = types.INIT
// 				path.EndpointA.SetChannel(channel)
// 			},
// 			expError: types.ErrInvalidChannelState,
// 		},
// 		{
// 			name: "channel not found",
// 			malleate: func() {
// 				path.EndpointA.Chain.DeleteKey(host.ChannelKey(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID))
// 			},
// 			expError: types.ErrChannelNotFound,
// 		},
// 		{
// 			name: "connection not found",
// 			malleate: func() {
// 				channel := path.EndpointA.GetChannel()
// 				channel.ConnectionHops = []string{"connection-100"}
// 				path.EndpointA.SetChannel(channel)
// 			},
// 			expError: connectiontypes.ErrConnectionNotFound,
// 		},
// 		{
// 			name: "counterparty upgrade sequence less than current sequence",
// 			malleate: func() {
// 				var ok bool
// 				errorReceipt, ok = suite.chainB.GetSimApp().IBCKeeper.ChannelKeeper.GetUpgradeErrorReceipt(suite.chainB.GetContext(), path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
// 				suite.Require().True(ok)

// 				// the channel sequence will be 1
// 				errorReceipt.Sequence = 0

// 				suite.chainB.GetSimApp().IBCKeeper.ChannelKeeper.SetUpgradeErrorReceipt(suite.chainB.GetContext(), path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, errorReceipt)

// 				suite.coordinator.CommitBlock(suite.chainB)
// 				suite.Require().NoError(path.EndpointA.UpdateClient())

// 				upgradeErrorReceiptKey := host.ChannelUpgradeErrorKey(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
// 				errorReceiptProof, proofHeight = suite.chainB.QueryProof(upgradeErrorReceiptKey)
// 			},
// 			expError: types.ErrInvalidUpgradeSequence,
// 		},
// 	}

// 	for _, tc := range tests {
// 		tc := tc
// 		suite.Run(tc.name, func() {
// 			suite.SetupTest()

// 			path = ibctesting.NewPath(suite.chainA, suite.chainB)
// 			suite.coordinator.Setup(path)

// 			path.EndpointA.ChannelConfig.ProposedUpgrade.Fields.Version = mock.UpgradeVersion
// 			path.EndpointB.ChannelConfig.ProposedUpgrade.Fields.Version = mock.UpgradeVersion

// 			suite.Require().NoError(path.EndpointA.ChanUpgradeInit())

// 			// cause the upgrade to fail on chain b so an error receipt is written.
// 			suite.chainB.GetSimApp().IBCMockModule.IBCApp.OnChanUpgradeTry = func(
// 				ctx sdk.Context, portID, channelID string, order types.Order, connectionHops []string, counterpartyVersion string,
// 			) (string, error) {
// 				return "", fmt.Errorf("mock app callback failed")
// 			}

// 			suite.Require().NoError(path.EndpointB.ChanUpgradeTry())

// 			suite.Require().NoError(path.EndpointA.UpdateClient())

// 			upgradeErrorReceiptKey := host.ChannelUpgradeErrorKey(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
// 			errorReceiptProof, proofHeight = suite.chainB.QueryProof(upgradeErrorReceiptKey)

// 			var ok bool
// 			errorReceipt, ok = suite.chainB.GetSimApp().IBCKeeper.ChannelKeeper.GetUpgradeErrorReceipt(suite.chainB.GetContext(), path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
// 			suite.Require().True(ok)

// 			tc.malleate()

// 			err := suite.chainA.GetSimApp().IBCKeeper.ChannelKeeper.ChanUpgradeCancel(suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, errorReceipt, errorReceiptProof, proofHeight)

// 			expPass := tc.expError == nil
// 			if expPass {
// 				suite.Require().NoError(err)
// 			} else {
// 				suite.Require().ErrorIs(err, tc.expError)
// 			}
// 		})
// 	}
// }
//
// func (suite *KeeperTestSuite) TestWriteUpgradeCancelChannel() {
//	suite.SetupTest()
//
//	path := ibctesting.NewPath(suite.chainA, suite.chainB)
//	suite.coordinator.Setup(path)
//
//	path.EndpointA.ChannelConfig.ProposedUpgrade.Fields.Version = mock.UpgradeVersion
//	path.EndpointB.ChannelConfig.ProposedUpgrade.Fields.Version = mock.UpgradeVersion
//
//	suite.Require().NoError(path.EndpointA.ChanUpgradeInit())
//
//	// cause the upgrade to fail on chain b so an error receipt is written.
//	suite.chainB.GetSimApp().IBCMockModule.IBCApp.OnChanUpgradeTry = func(
//		ctx sdk.Context, portID, channelID string, order types.Order, connectionHops []string, counterpartyVersion string,
//	) (string, error) {
//		return "", fmt.Errorf("mock app callback failed")
//	}
//
//	err := path.EndpointB.ChanUpgradeTry()
//	suite.Require().NoError(err)
//
//	err = path.EndpointA.UpdateClient()
//	suite.Require().NoError(err)
//
//	errorReceipt, ok := suite.chainB.GetSimApp().IBCKeeper.ChannelKeeper.GetUpgradeErrorReceipt(suite.chainB.GetContext(), path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
//	suite.Require().True(ok)
//
//	ctx := suite.chainA.GetContext()
//	suite.chainA.GetSimApp().IBCKeeper.ChannelKeeper.WriteUpgradeCancelChannel(ctx, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, errorReceipt.Sequence)
//
//	channel := path.EndpointA.GetChannel()
//
//	// Verify that channel has been restored to previous state
//	suite.Require().Equal(types.OPEN, channel.State)
//	suite.Require().Equal(types.NOTINFLUSH, channel.FlushStatus)
//	suite.Require().Equal(mock.Version, channel.Version)
//	suite.Require().Equal(errorReceipt.Sequence, channel.UpgradeSequence)
//
//	// Assert that state stored for upgrade has been deleted
//	upgrade, found := suite.chainA.GetSimApp().IBCKeeper.ChannelKeeper.GetUpgrade(suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
//	suite.Require().Equal(types.Upgrade{}, upgrade)
//	suite.Require().False(found)
//
//	lastPacketSequence, found := suite.chainA.GetSimApp().IBCKeeper.ChannelKeeper.GetCounterpartyLastPacketSequence(suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
//
//	// we need to find the event values from the proposed upgrade as the actual upgrade has been deleted.
//	proposedUpgrade := path.EndpointA.GetProposedUpgrade()
//	events := ctx.EventManager().Events().ToABCIEvents()
//	expEvents := ibctesting.EventsMap{
//		types.EventTypeChannelUpgradeCancel: {
//			types.AttributeKeyPortID:                path.EndpointA.ChannelConfig.PortID,
//			types.AttributeKeyChannelID:             path.EndpointA.ChannelID,
//			types.AttributeCounterpartyPortID:       path.EndpointB.ChannelConfig.PortID,
//			types.AttributeCounterpartyChannelID:    path.EndpointB.ChannelID,
//			types.AttributeKeyUpgradeConnectionHops: proposedUpgrade.Fields.ConnectionHops[0],
//			types.AttributeKeyUpgradeVersion:        proposedUpgrade.Fields.Version,
//			types.AttributeKeyUpgradeOrdering:       proposedUpgrade.Fields.Ordering.String(),
//			types.AttributeKeyUpgradeSequence:       fmt.Sprintf("%d", channel.UpgradeSequence),
//		},
//		sdk.EventTypeMessage: {
//			sdk.AttributeKeyModule: types.AttributeValueCategory,
//		},
//	}
//
//	suite.Require().Equal(uint64(0), lastPacketSequence)
//	suite.Require().False(found)
//	ibctesting.AssertEvents(&suite.Suite, expEvents, events)
//
//	counterpartyUpgrade, found := suite.chainA.GetSimApp().IBCKeeper.ChannelKeeper.GetCounterpartyUpgrade(suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
//	suite.Require().Equal(types.Upgrade{}, counterpartyUpgrade)
//	suite.Require().False(found)
// }

// func (suite *KeeperTestSuite) TestChanUpgradeTimeout() {
// 	var (
// 		path                     *ibctesting.Path
// 		errReceipt               *types.ErrorReceipt
// 		proofHeight              exported.Height
// 		proofCounterpartyChannel []byte
// 		proofErrorReceipt        []byte
// 	)

// 	testCases := []struct {
// 		name     string
// 		malleate func()
// 		expError error
// 	}{
// 		// {
// 		// 	"success: proof height has passed",
// 		// 	func() {},
// 		// 	nil,
// 		// },
// 		{
// 			"success: proof timestamp has passed",
// 			func() {
// 				upgrade := path.EndpointA.GetProposedUpgrade()
// 				upgrade.Timeout.Height = defaultTimeoutHeight
// 				upgrade.Timeout.Timestamp = 5
// 				suite.chainA.GetSimApp().IBCKeeper.ChannelKeeper.SetUpgrade(suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, upgrade)

// 				suite.Require().NoError(path.EndpointA.UpdateClient())

// 				proofCounterpartyChannel, _, proofHeight = path.EndpointA.QueryChannelUpgradeProof()
// 				upgradeErrorReceiptKey := host.ChannelUpgradeErrorKey(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
// 				proofErrorReceipt, _ = suite.chainB.QueryProof(upgradeErrorReceiptKey)
// 			},
// 			nil,
// 		},
// 		{
// 			"success: non-nil error receipt",
// 			func() {
// 				errReceipt = &types.ErrorReceipt{
// 					Sequence: 0,
// 					Message:  types.ErrInvalidUpgrade.Error(),
// 				}

// 				suite.chainB.GetSimApp().IBCKeeper.ChannelKeeper.SetUpgradeErrorReceipt(suite.chainB.GetContext(), path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, *errReceipt)

// 				suite.Require().NoError(path.EndpointB.UpdateClient())
// 				suite.Require().NoError(path.EndpointA.UpdateClient())

// 				proofCounterpartyChannel, _, proofHeight = path.EndpointA.QueryChannelUpgradeProof()
// 				upgradeErrorReceiptKey := host.ChannelUpgradeErrorKey(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
// 				proofErrorReceipt, _ = suite.chainB.QueryProof(upgradeErrorReceiptKey)
// 			},
// 			nil,
// 		},
// 		{
// 			"channel not found",
// 			func() {
// 				path.EndpointA.ChannelID = ibctesting.InvalidID
// 			},
// 			types.ErrChannelNotFound,
// 		},
// 		{
// 			"channel state is not in INITUPGRADE state",
// 			func() {
// 				suite.Require().NoError(path.EndpointA.SetChannelState(types.ACKUPGRADE))
// 			},
// 			types.ErrInvalidChannelState,
// 		},
// 		{
// 			"current upgrade not found",
// 			func() {
// 				suite.chainA.DeleteKey(host.ChannelUpgradeKey(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID))
// 			},
// 			types.ErrUpgradeNotFound,
// 		},
// 		{
// 			"connection not found",
// 			func() {
// 				channel := path.EndpointA.GetChannel()
// 				channel.ConnectionHops[0] = ibctesting.InvalidID
// 				path.EndpointA.SetChannel(channel)
// 			},
// 			connectiontypes.ErrConnectionNotFound,
// 		},
// 		{
// 			"connection not open",
// 			func() {
// 				connectionEnd := path.EndpointA.GetConnection()
// 				connectionEnd.State = connectiontypes.UNINITIALIZED
// 				path.EndpointA.SetConnection(connectionEnd)
// 			},
// 			connectiontypes.ErrInvalidConnectionState,
// 		},
// 		{
// 			"unable to retrieve timestamp at proof height",
// 			func() {
// 				proofHeight = suite.chainA.GetTimeoutHeight()
// 			},
// 			clienttypes.ErrConsensusStateNotFound,
// 		},
// 		{
// 			"timeout has not passed",
// 			func() {
// 				upgrade := path.EndpointA.GetProposedUpgrade()
// 				upgrade.Timeout.Height = suite.chainA.GetTimeoutHeight()
// 				suite.chainA.GetSimApp().IBCKeeper.ChannelKeeper.SetUpgrade(suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, upgrade)

// 				suite.Require().NoError(path.EndpointA.UpdateClient())

// 				proofCounterpartyChannel, _, proofHeight = path.EndpointA.QueryChannelUpgradeProof()
// 				upgradeErrorReceiptKey := host.ChannelUpgradeErrorKey(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
// 				proofErrorReceipt, _ = suite.chainB.QueryProof(upgradeErrorReceiptKey)
// 			},
// 			types.ErrInvalidUpgradeTimeout,
// 		},
// 		{
// 			"counterparty channel state is not OPEN or INITUPGRADE (crossing hellos)",
// 			func() {
// 				channel := path.EndpointB.GetChannel()
// 				channel.State = types.TRYUPGRADE
// 				path.EndpointB.SetChannel(channel)

// 				suite.Require().NoError(path.EndpointB.UpdateClient())
// 				suite.Require().NoError(path.EndpointA.UpdateClient())

// 				proofCounterpartyChannel, _, proofHeight = path.EndpointA.QueryChannelUpgradeProof()
// 				upgradeErrorReceiptKey := host.ChannelUpgradeErrorKey(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
// 				proofErrorReceipt, _ = suite.chainB.QueryProof(upgradeErrorReceiptKey)
// 			},
// 			types.ErrInvalidChannelState,
// 		},
// 		{
// 			"non-nil error receipt: error receipt seq greater than current upgrade seq",
// 			func() {
// 				errReceipt = &types.ErrorReceipt{
// 					Sequence: 3,
// 					Message:  types.ErrInvalidUpgrade.Error(),
// 				}
// 			},
// 			types.ErrInvalidUpgradeSequence,
// 		},
// 		{
// 			"non-nil error receipt: error receipt seq equal to current upgrade seq",
// 			func() {
// 				errReceipt = &types.ErrorReceipt{
// 					Sequence: 1,
// 					Message:  types.ErrInvalidUpgrade.Error(),
// 				}
// 			},
// 			types.ErrInvalidUpgradeSequence,
// 		},
// 	}

// 	for _, tc := range testCases {
// 		tc := tc
// 		suite.Run(tc.name, func() {
// 			suite.SetupTest()
// 			expPass := tc.expError == nil

// 			path = ibctesting.NewPath(suite.chainA, suite.chainB)
// 			suite.coordinator.Setup(path)

// 			path.EndpointA.ChannelConfig.ProposedUpgrade.Fields.Version = mock.UpgradeVersion
// 			path.EndpointB.ChannelConfig.ProposedUpgrade.Fields.Version = mock.UpgradeVersion

// 			errReceipt = nil

// 			// set timeout height to 1 to ensure timeout
// 			path.EndpointA.ChannelConfig.ProposedUpgrade.Timeout.Height = clienttypes.NewHeight(1, 1)
// 			suite.Require().NoError(path.EndpointA.ChanUpgradeInit())

// 			// ensure clients are up to date to receive valid proofs
// 			suite.Require().NoError(path.EndpointB.UpdateClient())
// 			suite.Require().NoError(path.EndpointA.UpdateClient())

// 			proofCounterpartyChannel, _, proofHeight = path.EndpointA.QueryChannelUpgradeProof()
// 			upgradeErrorReceiptKey := host.ChannelUpgradeErrorKey(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
// 			proofErrorReceipt, _ = suite.chainB.QueryProof(upgradeErrorReceiptKey)

// 			tc.malleate()

// 			err := suite.chainA.GetSimApp().IBCKeeper.ChannelKeeper.ChanUpgradeTimeout(
// 				suite.chainA.GetContext(),
// 				path.EndpointA.ChannelConfig.PortID,
// 				path.EndpointA.ChannelID,
// 				path.EndpointB.GetChannel(),
// 				errReceipt,
// 				proofCounterpartyChannel,
// 				proofErrorReceipt,
// 				proofHeight,
// 			)

// 			if expPass {
// 				suite.Require().NoError(err)
// 			} else {
// 				suite.assertUpgradeError(err, tc.expError)
// 			}
// 		})
// 	}
// }

func (suite *KeeperTestSuite) TestStartFlush() {
	var path *ibctesting.Path

	testCases := []struct {
		name     string
		malleate func()
		expError error
	}{
		{
			"success",
			func() {},
			nil,
		},
		{
			"channel not found",
			func() {
				path.EndpointB.ChannelID = "invalid-channel"
				path.EndpointB.ChannelConfig.PortID = "invalid-port"
			},
			types.ErrChannelNotFound,
		},
		{
			"connection not found",
			func() {
				channel := path.EndpointB.GetChannel()
				channel.ConnectionHops[0] = ibctesting.InvalidID
				path.EndpointB.SetChannel(channel)
			},
			connectiontypes.ErrConnectionNotFound,
		},
		{
			"connection state is not in OPEN state",
			func() {
				conn := path.EndpointB.GetConnection()
				conn.State = connectiontypes.INIT
				path.EndpointB.SetConnection(conn)
			},
			connectiontypes.ErrInvalidConnectionState,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTest()

			path = ibctesting.NewPath(suite.chainA, suite.chainB)
			suite.coordinator.Setup(path)

			path.EndpointA.ChannelConfig.ProposedUpgrade.Fields.Version = mock.UpgradeVersion
			path.EndpointB.ChannelConfig.ProposedUpgrade.Fields.Version = mock.UpgradeVersion

			err := path.EndpointA.ChanUpgradeInit()
			suite.Require().NoError(err)

			// crossing hellos so that the upgrade is created on chain B.
			// the ChanUpgradeInit sub protocol is also called when it is not a crossing hello situation.
			err = path.EndpointB.ChanUpgradeInit()
			suite.Require().NoError(err)

			tc.malleate()

			err = suite.chainB.GetSimApp().IBCKeeper.ChannelKeeper.StartFlushing(
				suite.chainB.GetContext(), path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID,
			)

			if tc.expError != nil {
				suite.assertUpgradeError(err, tc.expError)
			} else {
				channel := path.EndpointB.GetChannel()
				upgrade := path.EndpointB.GetChannelUpgrade()

				nextSequenceSend, ok := suite.chainB.GetSimApp().IBCKeeper.ChannelKeeper.GetNextSequenceSend(suite.chainB.GetContext(), path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
				suite.Require().True(ok)

				suite.Require().Equal(types.STATE_FLUSHING, channel.State)
				suite.Require().Equal(nextSequenceSend-1, upgrade.LatestSequenceSend)

				// TODO: fix in https://github.com/cosmos/ibc-go/issues/4313
				suite.Require().Equal(types.NewTimeout(clienttypes.NewHeight(1, 1000), 0), upgrade.Timeout)
				suite.Require().NoError(err)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestValidateUpgradeFields() {
	var (
		proposedUpgrade *types.UpgradeFields
		path            *ibctesting.Path
	)
	tests := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			name: "change channel version",
			malleate: func() {
				proposedUpgrade.Version = mock.UpgradeVersion
			},
			expPass: true,
		},
		{
			name: "change connection hops",
			malleate: func() {
				path := ibctesting.NewPath(suite.chainA, suite.chainB)
				suite.coordinator.Setup(path)
				proposedUpgrade.ConnectionHops = []string{path.EndpointA.ConnectionID}
			},
			expPass: true,
		},
		{
			name:     "fails with unmodified fields",
			malleate: func() {},
			expPass:  false,
		},
		{
			name: "fails when connection is not set",
			malleate: func() {
				storeKey := suite.chainA.GetSimApp().GetKey(exported.StoreKey)
				kvStore := suite.chainA.GetContext().KVStore(storeKey)
				kvStore.Delete(host.ConnectionKey(ibctesting.FirstConnectionID))
			},
			expPass: false,
		},
		{
			name: "fails when connection is not open",
			malleate: func() {
				connection := path.EndpointA.GetConnection()
				connection.State = connectiontypes.UNINITIALIZED
				path.EndpointA.SetConnection(connection)
			},
			expPass: false,
		},
		{
			name: "fails when connection versions do not exist",
			malleate: func() {
				// update channel version first so that existing channel end is not identical to proposed upgrade
				proposedUpgrade.Version = mock.UpgradeVersion

				connection := path.EndpointA.GetConnection()
				connection.Versions = []*connectiontypes.Version{}
				path.EndpointA.SetConnection(connection)
			},
			expPass: false,
		},
		{
			name: "fails when connection version does not support the new ordering",
			malleate: func() {
				// update channel version first so that existing channel end is not identical to proposed upgrade
				proposedUpgrade.Version = mock.UpgradeVersion

				connection := path.EndpointA.GetConnection()
				connection.Versions = []*connectiontypes.Version{
					connectiontypes.NewVersion("1", []string{"ORDER_ORDERED"}),
				}
				path.EndpointA.SetConnection(connection)
			},
			expPass: false,
		},
	}

	for _, tc := range tests {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTest()
			path = ibctesting.NewPath(suite.chainA, suite.chainB)
			suite.coordinator.Setup(path)

			existingChannel := path.EndpointA.GetChannel()
			proposedUpgrade = &types.UpgradeFields{
				Ordering:       existingChannel.Ordering,
				ConnectionHops: existingChannel.ConnectionHops,
				Version:        existingChannel.Version,
			}

			tc.malleate()

			err := suite.chainA.GetSimApp().IBCKeeper.ChannelKeeper.ValidateSelfUpgradeFields(suite.chainA.GetContext(), *proposedUpgrade, existingChannel)
			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *KeeperTestSuite) assertUpgradeError(actualError, expError error) {
	suite.Require().Error(actualError)

	if expUpgradeError, ok := expError.(*types.UpgradeError); ok {
		upgradeError, ok := actualError.(*types.UpgradeError)
		suite.Require().True(ok)
		suite.Require().Equal(expUpgradeError.GetErrorReceipt(), upgradeError.GetErrorReceipt())
	}

	suite.Require().True(errorsmod.IsOf(actualError, expError), fmt.Sprintf("expected error: %s, actual error: %s", expError, actualError))
}

// TestAbortHandshake tests that when the channel handshake is aborted, the channel state
// is restored the previous state and that an error receipt is written, and upgrade state which
// is no longer required is deleted.
func (suite *KeeperTestSuite) TestAbortHandshake() {
	var (
		path         *ibctesting.Path
		upgradeError error
	)

	tests := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			name:     "success",
			malleate: func() {},
			expPass:  true,
		},
		{
			name: "regular error",
			malleate: func() {
				// in app callbacks error receipts should still be written if a regular error is returned.
				// i.e. not an instance of `types.UpgradeError`
				upgradeError = types.ErrInvalidUpgrade
			},
			expPass: true,
		},
		{
			name: "upgrade does not exist",
			malleate: func() {
				suite.chainA.DeleteKey(host.ChannelUpgradeKey(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID))
			},
			expPass: false,
		},
		{
			name: "channel does not exist",
			malleate: func() {
				suite.chainA.DeleteKey(host.ChannelKey(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID))
			},
			expPass: false,
		},
		{
			name: "fails with nil upgrade error",
			malleate: func() {
				upgradeError = nil
			},
			expPass: false,
		},
	}

	for _, tc := range tests {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTest()
			path = ibctesting.NewPath(suite.chainA, suite.chainB)
			suite.coordinator.Setup(path)

			channelKeeper := suite.chainA.GetSimApp().IBCKeeper.ChannelKeeper

			path.EndpointA.ChannelConfig.Version = mock.UpgradeVersion
			suite.Require().NoError(path.EndpointA.ChanUpgradeInit())

			// fetch the upgrade before abort for assertions later on.
			actualUpgrade, ok := channelKeeper.GetUpgrade(suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
			suite.Require().True(ok, "upgrade should be found")

			upgradeError = types.NewUpgradeError(1, types.ErrInvalidChannel)

			tc.malleate()

			if tc.expPass {
				suite.Require().NotPanics(func() {
					channelKeeper.MustAbortUpgrade(suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, upgradeError)
				})

				channel, found := channelKeeper.GetChannel(suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
				suite.Require().True(found, "channel should be found")

				suite.Require().Equal(types.OPEN, channel.State, "channel state should be %s", types.OPEN.String())
				suite.Require().Equal(types.NOTINFLUSH, channel.FlushStatus, "channel flush status should be %s", types.NOTINFLUSH.String())
				_, found = channelKeeper.GetUpgrade(suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
				suite.Require().False(found, "upgrade info should be deleted")

				errorReceipt, found := channelKeeper.GetUpgradeErrorReceipt(suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
				suite.Require().True(found, "error receipt should be found")

				if ue, ok := upgradeError.(*types.UpgradeError); ok {
					suite.Require().Equal(ue.GetErrorReceipt(), errorReceipt, "error receipt does not match expected error receipt")
				}

				_, found = channelKeeper.GetCounterpartyLastPacketSequence(suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
				suite.Require().False(found, "counterparty last packet sequence should not be found")

			} else {

				suite.Require().Panics(func() {
					channelKeeper.MustAbortUpgrade(suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, upgradeError)
				})

				channel, found := channelKeeper.GetChannel(suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
				if found { // test cases uses a channel that exists
					suite.Require().Equal(types.OPEN, channel.State, "channel state should not be restored to %s", types.OPEN.String())
				}

				_, found = channelKeeper.GetUpgradeErrorReceipt(suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
				suite.Require().False(found, "error receipt should not be found")

				upgrade, found := channelKeeper.GetUpgrade(suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
				if found { // this should be all test cases except for when the upgrade is explicitly deleted.
					suite.Require().Equal(actualUpgrade, upgrade, "upgrade info should not be deleted")
				}

				// TODO: assertion that GetCounterpartyLastPacketSequence is present and correct
			}
		})
	}
}

func (suite *KeeperTestSuite) TestCheckForUpgradeCompatibility() {
	var (
		path                      *ibctesting.Path
		upgradeFields             types.UpgradeFields
		counterpartyUpgradeFields types.UpgradeFields
	)

	testCases := []struct {
		name     string
		malleate func()
		expError error
	}{
		{
			"success",
			func() {},
			nil,
		},
		{
			"upgrade ordering is not the same on both sides",
			func() {
				upgradeFields.Ordering = types.ORDERED
			},
			types.ErrIncompatibleCounterpartyUpgrade,
		},
		{
			"proposed connection is not found",
			func() {
				upgradeFields.ConnectionHops[0] = ibctesting.InvalidID
			},
			connectiontypes.ErrConnectionNotFound,
		},
		{
			"proposed connection is not in OPEN state",
			func() {
				// reuse existing connection to create a new connection in a non OPEN state
				connectionEnd := path.EndpointB.GetConnection()
				connectionEnd.State = connectiontypes.UNINITIALIZED
				connectionEnd.Counterparty.ConnectionId = counterpartyUpgradeFields.ConnectionHops[0] // both sides must be each other's counterparty

				// set proposed connection in state
				proposedConnectionID := "connection-100"
				suite.chainB.GetSimApp().GetIBCKeeper().ConnectionKeeper.SetConnection(suite.chainB.GetContext(), proposedConnectionID, connectionEnd)
				upgradeFields.ConnectionHops[0] = proposedConnectionID
			},
			connectiontypes.ErrInvalidConnectionState,
		},
		{
			"proposed connection ends are not each other's counterparty",
			func() {
				// reuse existing connection to create a new connection in a non OPEN state
				connectionEnd := path.EndpointB.GetConnection()
				// ensure counterparty connectionID does not match connectionID set in counterparty proposed upgrade
				connectionEnd.Counterparty.ConnectionId = "connection-50"

				// set proposed connection in state
				proposedConnectionID := "connection-100"
				suite.chainB.GetSimApp().GetIBCKeeper().ConnectionKeeper.SetConnection(suite.chainB.GetContext(), proposedConnectionID, connectionEnd)
				upgradeFields.ConnectionHops[0] = proposedConnectionID
			},
			types.ErrIncompatibleCounterpartyUpgrade,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTest()

			path = ibctesting.NewPath(suite.chainA, suite.chainB)
			suite.coordinator.Setup(path)

			path.EndpointA.ChannelConfig.ProposedUpgrade.Fields.Version = mock.UpgradeVersion
			path.EndpointB.ChannelConfig.ProposedUpgrade.Fields.Version = mock.UpgradeVersion

			err := path.EndpointA.ChanUpgradeInit()
			suite.Require().NoError(err)

			upgradeFields = path.EndpointA.GetProposedUpgrade().Fields
			counterpartyUpgradeFields = path.EndpointB.GetProposedUpgrade().Fields

			tc.malleate()

			err = suite.chainB.GetSimApp().IBCKeeper.ChannelKeeper.CheckForUpgradeCompatibility(suite.chainB.GetContext(), upgradeFields, counterpartyUpgradeFields)
			if tc.expError != nil {
				suite.Require().ErrorIs(err, tc.expError)
			} else {
				suite.Require().NoError(err)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestSyncUpgradeSequence() {
	var (
		path                        *ibctesting.Path
		counterpartyUpgradeSequence uint64
	)

	testCases := []struct {
		name     string
		malleate func()
		expError error
	}{
		{
			"success",
			func() {},
			nil,
		},
		{
			"upgrade sequence mismatch, endpointB channel upgrade sequence is ahead",
			func() {
				channel := path.EndpointB.GetChannel()
				channel.UpgradeSequence = 10
				path.EndpointB.SetChannel(channel)
			},
			types.NewUpgradeError(10, types.ErrInvalidUpgradeSequence), // max sequence will be returned
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTest()

			path = ibctesting.NewPath(suite.chainA, suite.chainB)
			suite.coordinator.Setup(path)

			path.EndpointA.ChannelConfig.ProposedUpgrade.Fields.Version = mock.UpgradeVersion
			path.EndpointB.ChannelConfig.ProposedUpgrade.Fields.Version = mock.UpgradeVersion

			err := path.EndpointA.ChanUpgradeInit()
			suite.Require().NoError(err)

			err = path.EndpointB.ChanUpgradeInit()
			suite.Require().NoError(err)

			counterpartyUpgradeSequence = 1

			tc.malleate()

			err = suite.chainB.GetSimApp().IBCKeeper.ChannelKeeper.SyncUpgradeSequence(suite.chainB.GetContext(), path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, path.EndpointB.GetChannel(), counterpartyUpgradeSequence)
			if tc.expError != nil {
				suite.Require().ErrorIs(err, tc.expError)
			} else {
				suite.Require().NoError(err)
			}
		})
	}
}
