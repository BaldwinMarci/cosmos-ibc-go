package types_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"
)

const (
	denom              = "transfer/gaiachannel/atom"
	amount             = "100"
	largeAmount        = "18446744073709551616"                                                           // one greater than largest uint64 (^uint64(0))
	invalidLargeAmount = "115792089237316195423570985008687907853269984665640564039457584007913129639936" // 2^256
)

// TestFungibleTokenPacketDataValidateBasic tests ValidateBasic for FungibleTokenPacketData
func TestFungibleTokenPacketDataValidateBasic(t *testing.T) {
	testCases := []struct {
		name       string
		packetData types.FungibleTokenPacketData
		expPass    bool
	}{
		{"valid packet", types.NewFungibleTokenPacketData(denom, amount, sender, receiver, ""), true},
		{"valid packet with memo", types.NewFungibleTokenPacketData(denom, amount, sender, receiver, "memo"), true},
		{"valid packet with large amount", types.NewFungibleTokenPacketData(denom, largeAmount, sender, receiver, ""), true},
		{"invalid denom", types.NewFungibleTokenPacketData("", amount, sender, receiver, ""), false},
		{"invalid empty amount", types.NewFungibleTokenPacketData(denom, "", sender, receiver, ""), false},
		{"invalid zero amount", types.NewFungibleTokenPacketData(denom, "0", sender, receiver, ""), false},
		{"invalid negative amount", types.NewFungibleTokenPacketData(denom, "-1", sender, receiver, ""), false},
		{"invalid large amount", types.NewFungibleTokenPacketData(denom, invalidLargeAmount, sender, receiver, ""), false},
		{"missing sender address", types.NewFungibleTokenPacketData(denom, amount, emptyAddr, receiver, ""), false},
		{"missing recipient address", types.NewFungibleTokenPacketData(denom, amount, sender, emptyAddr, ""), false},
	}

	for i, tc := range testCases {
		err := tc.packetData.ValidateBasic()
		if tc.expPass {
			require.NoError(t, err, "valid test case %d failed: %v", i, err)
		} else {
			require.Error(t, err, "invalid test case %d passed: %s", i, tc.name)
		}
	}
}

func (suite *TypesTestSuite) TestGetSourceCallbackAddress() {
	testCases := []struct {
		name       string
		packetData types.FungibleTokenPacketData
		expPass    bool
	}{
		{
			"memo is empty",
			types.FungibleTokenPacketData{
				Denom:    denom,
				Amount:   amount,
				Sender:   sender,
				Receiver: receiver,
				Memo:     "",
			},
			false,
		},
		{
			"memo is not json string",
			types.FungibleTokenPacketData{
				Denom:    denom,
				Amount:   amount,
				Sender:   sender,
				Receiver: receiver,
				Memo:     "memo",
			},
			false,
		},
		{
			"memo does not have callbacks in json struct",
			types.FungibleTokenPacketData{
				Denom:    denom,
				Amount:   amount,
				Sender:   sender,
				Receiver: receiver,
				Memo:     `{"Key": 10}`,
			},
			false,
		},
		{
			"memo has callbacks in json struct but does not have src_callback_address key",
			types.FungibleTokenPacketData{
				Denom:    denom,
				Amount:   amount,
				Sender:   sender,
				Receiver: receiver,
				Memo:     `{"callbacks": {"Key": 10}}`,
			},
			false,
		},
		{
			"memo has callbacks in json struct but does not have string value for src_callback_address key",
			types.FungibleTokenPacketData{
				Denom:    denom,
				Amount:   amount,
				Sender:   sender,
				Receiver: receiver,
				Memo:     `{"callbacks": {"src_callback_address": 10}}`,
			},
			false,
		},
		{
			"memo has callbacks in json struct and properly formatted src_callback_address which does not match packet sender",
			types.FungibleTokenPacketData{
				Denom:    denom,
				Amount:   amount,
				Sender:   sender,
				Receiver: receiver,
				Memo:     `{"callbacks": {"src_callback_address": "testAddress"}}`,
			},
			false,
		},
		{
			"valid src_callback_address specified in memo that matches sender",
			types.FungibleTokenPacketData{
				Denom:    denom,
				Amount:   amount,
				Sender:   sender,
				Receiver: receiver,
				Memo:     fmt.Sprintf(`{"callbacks": {"src_callback_address": "%s"}}`, sender),
			},
			true,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			srcCbAddr := tc.packetData.GetSourceCallbackAddress()

			if tc.expPass {
				suite.Require().Equal(sender, srcCbAddr)
			} else {
				suite.Require().Equal("", srcCbAddr)
			}
		})
	}
}

func (suite *TypesTestSuite) TestGetDestCallbackAddress() {
	testCases := []struct {
		name       string
		packetData types.FungibleTokenPacketData
		expPass    bool
	}{
		{
			"memo is empty",
			types.FungibleTokenPacketData{
				Denom:    denom,
				Amount:   amount,
				Sender:   sender,
				Receiver: receiver,
				Memo:     "",
			},
			false,
		},
		{
			"memo is not json string",
			types.FungibleTokenPacketData{
				Denom:    denom,
				Amount:   amount,
				Sender:   sender,
				Receiver: receiver,
				Memo:     "memo",
			},
			false,
		},
		{
			"memo does not have callbacks in json struct",
			types.FungibleTokenPacketData{
				Denom:    denom,
				Amount:   amount,
				Sender:   sender,
				Receiver: receiver,
				Memo:     `{"Key": 10}`,
			},
			false,
		},
		{
			"memo has callbacks in json struct but does not have dest_callback_address key",
			types.FungibleTokenPacketData{
				Denom:    denom,
				Amount:   amount,
				Sender:   sender,
				Receiver: receiver,
				Memo:     `{"callbacks": {"Key": 10}}`,
			},
			false,
		},
		{
			"memo has callbacks in json struct but does not have string value for dest_callback_address key",
			types.FungibleTokenPacketData{
				Denom:    denom,
				Amount:   amount,
				Sender:   sender,
				Receiver: receiver,
				Memo:     `{"callbacks": {"dest_callback_address": 10}}`,
			},
			false,
		},
		{
			"memo has callbacks in json struct and properly formatted dest_callback_address which does not match packet sender",
			types.FungibleTokenPacketData{
				Denom:    denom,
				Amount:   amount,
				Sender:   sender,
				Receiver: receiver,
				Memo:     `{"callbacks": {"dest_callback_address": "testAddress"}}`,
			},
			false,
		},
		{
			"valid dest_callback_address specified in memo that matches sender",
			types.FungibleTokenPacketData{
				Denom:    denom,
				Amount:   amount,
				Sender:   sender,
				Receiver: receiver,
				Memo:     fmt.Sprintf(`{"callbacks": {"dest_callback_address": "%s"}}`, receiver),
			},
			true,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			destCbAddr := tc.packetData.GetDestCallbackAddress()

			if tc.expPass {
				suite.Require().Equal(receiver, destCbAddr)
			} else {
				suite.Require().Equal("", destCbAddr)
			}
		})
	}
}

func (suite *TypesTestSuite) TestUserDefinedGasLimit() {
	packetData := types.FungibleTokenPacketData{
		Denom:    denom,
		Amount:   amount,
		Sender:   sender,
		Receiver: receiver,
		Memo:     `{"callbacks": {"user_defined_gas_limit": 100}}`,
	}

	suite.Require().Equal(uint64(0), packetData.UserDefinedGasLimit(), "user defined gas limit does not return 0")
}
