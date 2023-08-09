package ibccallbacks_test

import (
	"fmt"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	icacontrollertypes "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/controller/types"
	ibccallbacks "github.com/cosmos/ibc-go/v7/modules/apps/callbacks"
	"github.com/cosmos/ibc-go/v7/modules/apps/callbacks/types"
	transfertypes "github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	channelkeeper "github.com/cosmos/ibc-go/v7/modules/core/04-channel/keeper"
	channeltypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	porttypes "github.com/cosmos/ibc-go/v7/modules/core/05-port/types"
	ibcerrors "github.com/cosmos/ibc-go/v7/modules/core/errors"
	ibctesting "github.com/cosmos/ibc-go/v7/testing"
	ibcmock "github.com/cosmos/ibc-go/v7/testing/mock"
)

func (s *CallbacksTestSuite) TestNewIBCMiddleware() {
	testCases := []struct {
		name          string
		instantiateFn func()
		expError      error
	}{
		{
			"success",
			func() {
				_ = ibccallbacks.NewIBCMiddleware(ibcmock.IBCModule{}, s.chainA.GetSimApp().GetIBCKeeper().ChannelKeeper, s.chainA.GetSimApp().MockContractKeeper, maxCallbackGas)
			},
			nil,
		},
		{
			"panics with nil underlying app",
			func() {
				_ = ibccallbacks.NewIBCMiddleware(nil, s.chainA.GetSimApp().GetIBCKeeper().ChannelKeeper, s.chainA.GetSimApp().MockContractKeeper, maxCallbackGas)
			},
			fmt.Errorf("underlying application does not implement %T", (*types.CallbacksCompatibleModule)(nil)),
		},
		{
			"panics with nil contract keeper",
			func() {
				_ = ibccallbacks.NewIBCMiddleware(ibcmock.IBCModule{}, s.chainA.GetSimApp().GetIBCKeeper().ChannelKeeper, nil, maxCallbackGas)
			},
			fmt.Errorf("contract keeper cannot be nil"),
		},
		{
			"panics with nil ics4Wrapper",
			func() {
				_ = ibccallbacks.NewIBCMiddleware(ibcmock.IBCModule{}, nil, s.chainA.GetSimApp().MockContractKeeper, maxCallbackGas)
			},
			fmt.Errorf("ICS4Wrapper cannot be nil"),
		},
	}

	for _, tc := range testCases {
		tc := tc
		s.Run(tc.name, func() {
			s.setupChains()

			expPass := tc.expError == nil
			if expPass {
				s.Require().NotPanics(tc.instantiateFn, "unexpected panic: NewIBCMiddleware")
			} else {
				s.Require().PanicsWithError(tc.expError.Error(), tc.instantiateFn, "expected panic with error: ", tc.expError.Error())
			}
		})
	}
}

func (s *CallbacksTestSuite) TestWithICS4Wrapper() {
	s.setupChains()

	cbsMiddleware := ibccallbacks.IBCMiddleware{}
	s.Require().Nil(cbsMiddleware.GetICS4Wrapper())

	cbsMiddleware.WithICS4Wrapper(s.chainA.App.GetIBCKeeper().ChannelKeeper)
	ics4Wrapper := cbsMiddleware.GetICS4Wrapper()

	s.Require().IsType(channelkeeper.Keeper{}, ics4Wrapper)
}

func (s *CallbacksTestSuite) TestSendPacket() {
	var packetData transfertypes.FungibleTokenPacketData

	testCases := []struct {
		name            string
		malleate        func()
		callbackTrigger types.CallbackTrigger
		expError        error
	}{
		{
			"success",
			func() {},
			types.CallbackTriggerSendPacket,
			nil,
		},
		{
			"success: no-op on callback data is not valid",
			func() {
				packetData.Memo = `{"src_callback": {"address": ""}}`
			},
			"none", // improperly formatted callback data should result in no callback execution
			nil,
		},
		{
			"failure: ics4Wrapper SendPacket call fails",
			func() {
				s.path.EndpointA.ChannelID = "invalid-channel"
			},
			"none", // ics4wrapper  failure should result in no callback execution
			channeltypes.ErrChannelNotFound,
		},
		{
			"failure: callback execution fails, sender is not callback address",
			func() {
				packetData.Sender = ibcmock.MockCallbackUnauthorizedAddress
			},
			types.CallbackTriggerSendPacket,
			ibcmock.MockApplicationCallbackError, // execution failure on SendPacket should prevent packet sends
		},
	}

	for _, tc := range testCases {
		tc := tc
		s.Run(tc.name, func() {
			s.SetupTransferTest()

			// callbacks module is routed as top level middleware
			transferStack, ok := s.chainA.App.GetIBCKeeper().Router.GetRoute(transfertypes.ModuleName)
			s.Require().True(ok)

			packetData = transfertypes.NewFungibleTokenPacketData(
				ibctesting.TestCoin.GetDenom(), ibctesting.TestCoin.Amount.String(), callbackAddr,
				ibctesting.TestAccAddress, fmt.Sprintf(`{"src_callback": {"address": "%s"}}`, callbackAddr),
			)

			chanCap := s.path.EndpointA.Chain.GetChannelCapability(s.path.EndpointA.ChannelConfig.PortID, s.path.EndpointA.ChannelID)

			tc.malleate()

			seq, err := transferStack.(porttypes.Middleware).SendPacket(s.chainA.GetContext(), chanCap, s.path.EndpointA.ChannelConfig.PortID, s.path.EndpointA.ChannelID, s.chainB.GetTimeoutHeight(), 0, packetData.GetBytes())

			expPass := tc.expError == nil
			s.AssertHasExecutedExpectedCallback(tc.callbackTrigger, expPass)

			if expPass {
				s.Require().Nil(err)
				s.Require().Equal(uint64(1), seq)
			} else {
				s.Require().ErrorIs(tc.expError, err)
				s.Require().Equal(uint64(0), seq)
			}
		})
	}
}

func (s *CallbacksTestSuite) TestOnAcknowledgementPacket() {
	type expResult uint8
	const (
		noExecution expResult = iota
		callbackFailed
		callbackSuccess
	)

	var (
		packetData transfertypes.FungibleTokenPacketData
		packet     channeltypes.Packet
		ack        []byte
		ctx        sdk.Context
	)

	panicError := fmt.Errorf("panic error")

	testCases := []struct {
		name      string
		malleate  func()
		expResult expResult
		expError  error
	}{
		{
			"success",
			func() {},
			callbackSuccess,
			nil,
		},
		{
			"failure: underlying app OnAcknolwedgePacket fails",
			func() {
				ack = []byte("invalid ack")
			},
			noExecution,
			ibcerrors.ErrUnknownRequest,
		},
		{
			"success: no-op on callback data is not valid",
			func() {
				packetData.Memo = `{"src_callback": {"address": ""}}`
				packet.Data = packetData.GetBytes()
			},
			noExecution,
			nil,
		},
		{
			"failure: callback execution reach out of gas, but sufficient gas provided by relayer",
			func() {
				packetData.Memo = fmt.Sprintf(`{"src_callback": {"address":"%s", "gas_limit":"400000"}}`, callbackAddr)
				packet.Data = packetData.GetBytes()
			},
			callbackFailed,
			nil,
		},
		{
			"failure: callback execution panics on insufficient gas provided by relayer",
			func() {
				ctx = ctx.WithGasMeter(sdk.NewGasMeter(300_000))
			},
			callbackFailed,
			panicError,
		},
		{
			"failure: callback execution fails, unauthorized address",
			func() {
				packetData.Sender = ibcmock.MockCallbackUnauthorizedAddress
				packet.Data = packetData.GetBytes()
			},
			callbackFailed,
			nil, // execution failure in OnAcknowledgement should not block acknowledgement processing
		},
	}

	for _, tc := range testCases {
		tc := tc
		s.Run(tc.name, func() {
			s.SetupTransferTest()

			// set user gas limit above panic level in mock contract keeper
			packetData = transfertypes.NewFungibleTokenPacketData(
				ibctesting.TestCoin.GetDenom(), ibctesting.TestCoin.Amount.String(), callbackAddr, ibctesting.TestAccAddress,
				fmt.Sprintf(`{"src_callback": {"address":"%s", "gas_limit":"600000"}}`, callbackAddr),
			)

			packet = channeltypes.Packet{
				Sequence:           1,
				SourcePort:         s.path.EndpointA.ChannelConfig.PortID,
				SourceChannel:      s.path.EndpointA.ChannelID,
				DestinationPort:    s.path.EndpointB.ChannelConfig.PortID,
				DestinationChannel: s.path.EndpointB.ChannelID,
				Data:               packetData.GetBytes(),
				TimeoutHeight:      s.chainB.GetTimeoutHeight(),
				TimeoutTimestamp:   0,
			}

			ack = channeltypes.NewResultAcknowledgement([]byte{1}).Acknowledgement()
			ctx = s.chainA.GetContext()

			tc.malleate()

			// callbacks module is routed as top level middleware
			transferStack, ok := s.chainA.App.GetIBCKeeper().Router.GetRoute(transfertypes.ModuleName)
			s.Require().True(ok)

			onAcknowledgementPacket := func() error {
				return transferStack.OnAcknowledgementPacket(ctx, packet, ack, s.chainA.SenderAccount.GetAddress())
			}

			switch tc.expError {
			case nil:
				err := onAcknowledgementPacket()
				s.Require().Nil(err)

			case panicError:
				s.Require().PanicsWithValue(sdk.ErrorOutOfGas{
					Descriptor: fmt.Sprintf("mock %s callback panic", types.CallbackTriggerAcknowledgementPacket),
				}, func() {
					_ = onAcknowledgementPacket()
				})

			default:
				err := onAcknowledgementPacket()
				s.Require().ErrorIs(tc.expError, err)
			}

			sourceStatefulCounter := s.chainA.GetSimApp().MockContractKeeper.GetStateEntryCounter(s.chainA.GetContext())
			sourceCounters := s.chainA.GetSimApp().MockContractKeeper.Counters

			switch tc.expResult {
			case noExecution:
				s.Require().Len(sourceCounters, 0)
				s.Require().Equal(uint8(0), sourceStatefulCounter)

			case callbackFailed:
				s.Require().Len(sourceCounters, 1)
				s.Require().Equal(1, sourceCounters[types.CallbackTriggerAcknowledgementPacket])
				s.Require().Equal(uint8(0), sourceStatefulCounter)

			case callbackSuccess:
				s.Require().Len(sourceCounters, 1)
				s.Require().Equal(1, sourceCounters[types.CallbackTriggerAcknowledgementPacket])
				s.Require().Equal(uint8(1), sourceStatefulCounter)

			}
		})
	}
}

func (s *CallbacksTestSuite) TestUnmarshalPacketData() {
	s.setupChains()

	// We will pass the function call down the transfer stack to the transfer module
	// transfer stack UnmarshalPacketData call order: callbacks -> fee -> transfer
	transferStack, ok := s.chainA.App.GetIBCKeeper().Router.GetRoute(transfertypes.ModuleName)
	s.Require().True(ok)

	unmarshalerStack, ok := transferStack.(types.CallbacksCompatibleModule)
	s.Require().True(ok)

	expPacketData := transfertypes.FungibleTokenPacketData{
		Denom:    ibctesting.TestCoin.Denom,
		Amount:   ibctesting.TestCoin.Amount.String(),
		Sender:   ibctesting.TestAccAddress,
		Receiver: ibctesting.TestAccAddress,
		Memo:     fmt.Sprintf(`{"src_callback": {"address": "%s"}, "dest_callback": {"address":"%s"}}`, ibctesting.TestAccAddress, ibctesting.TestAccAddress),
	}
	data := expPacketData.GetBytes()

	packetData, err := unmarshalerStack.UnmarshalPacketData(data)
	s.Require().NoError(err)
	s.Require().Equal(expPacketData, packetData)
}

func (s *CallbacksTestSuite) TestGetAppVersion() {
	s.SetupICATest()

	// Obtain an IBC stack for testing. The function call will use the top of the stack which calls
	// directly to the channel keeper. Calling from a further down module in the stack is not necessary
	// for this test.
	icaControllerStack, ok := s.chainA.App.GetIBCKeeper().Router.GetRoute(icacontrollertypes.SubModuleName)
	s.Require().True(ok)

	controllerStack := icaControllerStack.(porttypes.Middleware)
	appVersion, found := controllerStack.GetAppVersion(s.chainA.GetContext(), s.path.EndpointA.ChannelConfig.PortID, s.path.EndpointA.ChannelID)
	s.Require().True(found)
	s.Require().Equal(s.path.EndpointA.ChannelConfig.Version, appVersion)
}

func (s *CallbacksTestSuite) TestOnChanCloseInit() {
	s.SetupICATest()

	// We will pass the function call down the icacontroller stack to the icacontroller module
	// icacontroller stack OnChanCloseInit call order: callbacks -> fee -> icacontroller
	icaControllerStack, ok := s.chainA.App.GetIBCKeeper().Router.GetRoute(icacontrollertypes.SubModuleName)
	s.Require().True(ok)

	controllerStack := icaControllerStack.(porttypes.Middleware)
	err := controllerStack.OnChanCloseInit(s.chainA.GetContext(), s.path.EndpointA.ChannelConfig.PortID, s.path.EndpointA.ChannelID)
	// we just check that this call is passed down to the icacontroller to return an error
	s.Require().ErrorIs(errorsmod.Wrap(ibcerrors.ErrInvalidRequest, "user cannot close channel"), err)
}

func (s *CallbacksTestSuite) TestOnChanCloseConfirm() {
	s.SetupICATest()

	// We will pass the function call down the icacontroller stack to the icacontroller module
	// icacontroller stack OnChanCloseConfirm call order: callbacks -> fee -> icacontroller
	icaControllerStack, ok := s.chainA.App.GetIBCKeeper().Router.GetRoute(icacontrollertypes.SubModuleName)
	s.Require().True(ok)

	controllerStack := icaControllerStack.(porttypes.Middleware)
	err := controllerStack.OnChanCloseConfirm(s.chainA.GetContext(), s.path.EndpointA.ChannelConfig.PortID, s.path.EndpointA.ChannelID)
	// we just check that this call is passed down to the icacontroller
	s.Require().NoError(err)
}

func (s *CallbacksTestSuite) TestWriteAcknowledgement() {
	s.SetupTransferTest()

	// build packet
	packetData := transfertypes.NewFungibleTokenPacketData(
		ibctesting.TestCoin.Denom,
		ibctesting.TestCoin.Amount.String(),
		ibctesting.TestAccAddress,
		ibctesting.TestAccAddress,
		fmt.Sprintf(`{"dest_callback": {"address":"%s"}}`, ibctesting.TestAccAddress),
	)

	packet := channeltypes.NewPacket(
		packetData.GetBytes(),
		1,
		s.path.EndpointA.ChannelConfig.PortID,
		s.path.EndpointA.ChannelID,
		s.path.EndpointB.ChannelConfig.PortID,
		s.path.EndpointB.ChannelID,
		clienttypes.NewHeight(1, 100),
		0,
	)

	transferStack, ok := s.chainB.App.GetIBCKeeper().Router.GetRoute(transfertypes.ModuleName)
	s.Require().True(ok)

	transferStackMw := transferStack.(porttypes.Middleware)

	ack := channeltypes.NewResultAcknowledgement([]byte("success"))
	chanCap := s.chainB.GetChannelCapability(s.path.EndpointB.ChannelConfig.PortID, s.path.EndpointB.ChannelID)

	err := transferStackMw.WriteAcknowledgement(s.chainB.GetContext(), chanCap, packet, ack)
	s.Require().NoError(err)

	packetAck, _ := s.chainB.GetSimApp().GetIBCKeeper().ChannelKeeper.GetPacketAcknowledgement(s.chainB.GetContext(), packet.DestinationPort, packet.DestinationChannel, 1)
	s.Require().Equal(packetAck, channeltypes.CommitAcknowledgement(ack.Acknowledgement()))
}

func (s *CallbacksTestSuite) TestWriteAcknowledgementError() {
	s.SetupICATest()

	packet := channeltypes.NewPacket(
		[]byte("invalid packet data"),
		1,
		s.path.EndpointA.ChannelConfig.PortID,
		s.path.EndpointA.ChannelID,
		"invalid_port",
		"invalid_channel",
		clienttypes.NewHeight(1, 100),
		0,
	)

	icaControllerStack, ok := s.chainB.App.GetIBCKeeper().Router.GetRoute(icacontrollertypes.SubModuleName)
	s.Require().True(ok)

	callbackStack := icaControllerStack.(porttypes.Middleware)

	ack := channeltypes.NewResultAcknowledgement([]byte("success"))
	chanCap := s.chainB.GetChannelCapability(s.path.EndpointB.ChannelConfig.PortID, s.path.EndpointB.ChannelID)

	err := callbackStack.WriteAcknowledgement(s.chainB.GetContext(), chanCap, packet, ack)
	s.Require().ErrorIs(err, errorsmod.Wrap(channeltypes.ErrChannelNotFound, packet.GetDestChannel()))
}

func (s *CallbacksTestSuite) TestOnTimeoutPacketError() {
	// The successful cases are tested in transfer_test.go and ica_test.go.
	// This test case tests the error case by passing an invalid packet data.
	s.SetupTransferTest()

	// We will pass the function call down the transfer stack to the transfer module
	// transfer stack OnTimeoutPacket call order: callbacks -> fee -> transfer
	transferStack, ok := s.chainA.App.GetIBCKeeper().Router.GetRoute(transfertypes.ModuleName)
	s.Require().True(ok)

	err := transferStack.OnTimeoutPacket(s.chainA.GetContext(), channeltypes.Packet{}, s.chainA.SenderAccount.GetAddress())
	s.Require().ErrorIs(ibcerrors.ErrUnknownRequest, err)
	s.Require().ErrorContains(err, "cannot unmarshal ICS-20 transfer packet data:")
}

func (s *CallbacksTestSuite) TestOnRecvPacketAsyncAck() {
	s.SetupMockFeeTest()

	module, _, err := s.chainA.App.GetIBCKeeper().PortKeeper.LookupModuleByPort(s.chainA.GetContext(), ibctesting.MockFeePort)
	s.Require().NoError(err)
	cbs, ok := s.chainA.App.GetIBCKeeper().Router.GetRoute(module)
	s.Require().True(ok)
	mockFeeCallbackStack, ok := cbs.(porttypes.Middleware)
	s.Require().True(ok)

	packet := channeltypes.NewPacket(
		ibcmock.MockAsyncPacketData,
		s.chainA.SenderAccount.GetSequence(),
		s.path.EndpointA.ChannelConfig.PortID,
		s.path.EndpointA.ChannelID,
		s.path.EndpointB.ChannelConfig.PortID,
		s.path.EndpointB.ChannelID,
		clienttypes.NewHeight(0, 100),
		0,
	)

	ack := mockFeeCallbackStack.OnRecvPacket(s.chainA.GetContext(), packet, s.chainA.SenderAccount.GetAddress())
	s.Require().Nil(ack)
	s.AssertHasExecutedExpectedCallback("none", true)
}

func (s *CallbacksTestSuite) TestOnRecvPacketFailedAck() {
	s.SetupMockFeeTest()

	module, _, err := s.chainA.App.GetIBCKeeper().PortKeeper.LookupModuleByPort(s.chainA.GetContext(), ibctesting.MockFeePort)
	s.Require().NoError(err)
	cbs, ok := s.chainA.App.GetIBCKeeper().Router.GetRoute(module)
	s.Require().True(ok)
	mockFeeCallbackStack, ok := cbs.(porttypes.Middleware)
	s.Require().True(ok)

	packet := channeltypes.NewPacket(
		nil,
		s.chainA.SenderAccount.GetSequence(),
		s.path.EndpointA.ChannelConfig.PortID,
		s.path.EndpointA.ChannelID,
		s.path.EndpointB.ChannelConfig.PortID,
		s.path.EndpointB.ChannelID,
		clienttypes.NewHeight(0, 100),
		0,
	)

	ack := mockFeeCallbackStack.OnRecvPacket(s.chainA.GetContext(), packet, s.chainA.SenderAccount.GetAddress())
	s.Require().Equal(ibcmock.MockFailAcknowledgement, ack)
	s.AssertHasExecutedExpectedCallback("none", true)
}

func (s *CallbacksTestSuite) TestOnRecvPacketLowRelayerGas() {
	s.SetupTransferTest()

	// build packet
	packetData := transfertypes.NewFungibleTokenPacketData(
		ibctesting.TestCoin.Denom,
		ibctesting.TestCoin.Amount.String(),
		ibctesting.TestAccAddress,
		ibctesting.TestAccAddress,
		fmt.Sprintf(`{"dest_callback": {"address":"%s", "gas_limit":"500000"}}`, ibctesting.TestAccAddress),
	)

	packet := channeltypes.NewPacket(
		packetData.GetBytes(),
		1,
		s.path.EndpointA.ChannelConfig.PortID,
		s.path.EndpointA.ChannelID,
		s.path.EndpointB.ChannelConfig.PortID,
		s.path.EndpointB.ChannelID,
		clienttypes.NewHeight(1, 100),
		0,
	)

	transferStack, ok := s.chainB.App.GetIBCKeeper().Router.GetRoute(transfertypes.ModuleName)
	s.Require().True(ok)

	transferStackMw := transferStack.(porttypes.Middleware)

	modifiedCtx := s.chainB.GetContext().WithGasMeter(sdk.NewGasMeter(400000))
	s.Require().PanicsWithValue(sdk.ErrorOutOfGas{
		Descriptor: fmt.Sprintf("mock %s callback panic", types.CallbackTriggerReceivePacket),
	}, func() {
		transferStackMw.OnRecvPacket(modifiedCtx, packet, s.chainB.SenderAccount.GetAddress())
	})

	// check that it doesn't panic when gas is high enough
	ack := transferStackMw.OnRecvPacket(s.chainB.GetContext(), packet, s.chainB.SenderAccount.GetAddress())
	s.Require().NotNil(ack)
}

func (s *CallbacksTestSuite) TestWriteAcknowledgementOogError() {
	s.SetupTransferTest()

	// build packet
	packetData := transfertypes.NewFungibleTokenPacketData(
		ibctesting.TestCoin.Denom,
		ibctesting.TestCoin.Amount.String(),
		ibctesting.TestAccAddress,
		ibctesting.TestAccAddress,
		fmt.Sprintf(`{"dest_callback": {"address":"%s", "gas_limit":"350000"}}`, ibctesting.TestAccAddress),
	)

	packet := channeltypes.NewPacket(
		packetData.GetBytes(),
		1,
		s.path.EndpointA.ChannelConfig.PortID,
		s.path.EndpointA.ChannelID,
		s.path.EndpointB.ChannelConfig.PortID,
		s.path.EndpointB.ChannelID,
		clienttypes.NewHeight(1, 100),
		0,
	)

	transferStack, ok := s.chainB.App.GetIBCKeeper().Router.GetRoute(transfertypes.ModuleName)
	s.Require().True(ok)

	transferStackMw := transferStack.(porttypes.Middleware)

	ack := channeltypes.NewResultAcknowledgement([]byte("success"))
	chanCap := s.chainB.GetChannelCapability(s.path.EndpointB.ChannelConfig.PortID, s.path.EndpointB.ChannelID)

	modifiedCtx := s.chainB.GetContext().WithGasMeter(sdk.NewGasMeter(300_000))
	s.Require().PanicsWithValue(sdk.ErrorOutOfGas{
		Descriptor: fmt.Sprintf("mock %s callback panic", types.CallbackTriggerReceivePacket),
	}, func() {
		_ = transferStackMw.WriteAcknowledgement(modifiedCtx, chanCap, packet, ack)
	})
}

func (s *CallbacksTestSuite) TestOnTimeoutPacketLowRelayerGas() {
	s.SetupTransferTest()

	timeoutHeight := clienttypes.GetSelfHeight(s.chainB.GetContext())
	timeoutTimestamp := uint64(s.chainB.GetContext().BlockTime().UnixNano())

	amount := ibctesting.TestCoin
	msg := transfertypes.NewMsgTransfer(
		s.path.EndpointA.ChannelConfig.PortID, s.path.EndpointA.ChannelID,
		amount, s.chainA.SenderAccount.GetAddress().String(),
		s.chainB.SenderAccount.GetAddress().String(), timeoutHeight, timeoutTimestamp,
		fmt.Sprintf(`{"src_callback": {"address":"%s", "gas_limit":"350000"}}`, ibctesting.TestAccAddress),
	)

	res, err := s.chainA.SendMsgs(msg)
	s.Require().NoError(err) // message committed

	packet, err := ibctesting.ParsePacketFromEvents(res.GetEvents().ToABCIEvents())
	s.Require().NoError(err) // packet committed
	s.Require().NotNil(packet)

	// need to update chainA's client representing chainB to prove missing ack
	err = s.path.EndpointA.UpdateClient()
	s.Require().NoError(err)

	transferStack, ok := s.chainA.App.GetIBCKeeper().Router.GetRoute(transfertypes.ModuleName)
	s.Require().True(ok)
	modifiedCtx := s.chainA.GetContext().WithGasMeter(sdk.NewGasMeter(300_000))
	s.Require().PanicsWithValue(sdk.ErrorOutOfGas{
		Descriptor: fmt.Sprintf("mock %s callback panic", types.CallbackTriggerTimeoutPacket),
	}, func() {
		_ = transferStack.OnTimeoutPacket(modifiedCtx, packet, s.chainA.SenderAccount.GetAddress())
	})
}
