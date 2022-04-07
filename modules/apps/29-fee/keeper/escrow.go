package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/cosmos/ibc-go/v3/modules/apps/29-fee/types"
	channeltypes "github.com/cosmos/ibc-go/v3/modules/core/04-channel/types"
)

// EscrowPacketFee sends the packet fee to the 29-fee module account to hold in escrow
func (k Keeper) EscrowPacketFee(ctx sdk.Context, packetID channeltypes.PacketId, packetFee types.PacketFee) error {
	if !k.IsFeeEnabled(ctx, packetID.PortId, packetID.ChannelId) {
		// users may not escrow fees on this channel. Must send packets without a fee message
		return sdkerrors.Wrap(types.ErrFeeNotEnabled, "cannot escrow fee for packet")
	}
	// check if the refund account exists
	refundAcc, err := sdk.AccAddressFromBech32(packetFee.RefundAddress)
	if err != nil {
		return err
	}

	hasRefundAcc := k.authKeeper.GetAccount(ctx, refundAcc)
	if hasRefundAcc == nil {
		return sdkerrors.Wrapf(types.ErrRefundAccNotFound, "account with address: %s not found", refundAcc)
	}

	coins := packetFee.Fee.Total()
	if err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, refundAcc, types.ModuleName, coins); err != nil {
		return err
	}

	fees := []types.PacketFee{packetFee}
	if feesInEscrow, found := k.GetFeesInEscrow(ctx, packetID); found {
		fees = append(fees, feesInEscrow.PacketFees...)
	}

	packetFees := types.NewPacketFees(fees)
	k.SetFeesInEscrow(ctx, packetID, packetFees)

	EmitIncentivizedPacket(ctx, packetID, packetFee)

	return nil
}

// DistributePacketFees pays the acknowledgement fee & receive fee for a given packetID while refunding the timeout fee to the refund account associated with the Fee.
func (k Keeper) DistributePacketFees(ctx sdk.Context, forwardRelayer string, reverseRelayer sdk.AccAddress, feesInEscrow []types.PacketFee) {
	forwardAddr, _ := sdk.AccAddressFromBech32(forwardRelayer)

	for _, packetFee := range feesInEscrow {
		refundAddr, err := sdk.AccAddressFromBech32(packetFee.RefundAddress)
		if err != nil {
			panic(fmt.Sprintf("could not parse refundAcc %s to sdk.AccAddress", packetFee.RefundAddress))
		}

		// distribute fee to valid forward relayer address otherwise refund the fee
		if !forwardAddr.Empty() && !k.bankKeeper.BlockedAddr(forwardAddr) {
			// distribute fee for forward relaying
			k.distributeFee(ctx, forwardAddr, packetFee.Fee.RecvFee)
		} else {
			// refund onRecv fee as forward relayer is not valid address
			k.distributeFee(ctx, refundAddr, packetFee.Fee.RecvFee)
		}

		// distribute fee for reverse relaying
		k.distributeFee(ctx, reverseRelayer, packetFee.Fee.AckFee)

		// refund timeout fee for unused timeout
		k.distributeFee(ctx, refundAddr, packetFee.Fee.TimeoutFee)
	}
}

// DistributePacketsFeesTimeout pays the timeout fee for a given packetID while refunding the acknowledgement fee & receive fee to the refund account associated with the Fee
func (k Keeper) DistributePacketFeesOnTimeout(ctx sdk.Context, timeoutRelayer sdk.AccAddress, feesInEscrow []types.PacketFee) {
	for _, feeInEscrow := range feesInEscrow {
		// check if refundAcc address works
		refundAddr, err := sdk.AccAddressFromBech32(feeInEscrow.RefundAddress)
		if err != nil {
			panic(fmt.Sprintf("could not parse refundAcc %s to sdk.AccAddress", feeInEscrow.RefundAddress))
		}

		// refund receive fee for unused forward relaying
		k.distributeFee(ctx, refundAddr, feeInEscrow.Fee.RecvFee)

		// refund ack fee for unused reverse relaying
		k.distributeFee(ctx, refundAddr, feeInEscrow.Fee.AckFee)

		// distribute fee for timeout relaying
		k.distributeFee(ctx, timeoutRelayer, feeInEscrow.Fee.TimeoutFee)
	}
}

// distributeFee will attempt to distribute the escrowed fee to the receiver address.
// If the distribution fails for any reason (such as the receiving address being blocked),
// the state changes will be discarded.
func (k Keeper) distributeFee(ctx sdk.Context, receiver sdk.AccAddress, fee sdk.Coins) {
	// cache context before trying to distribute fees
	cacheCtx, writeFn := ctx.CacheContext()

	err := k.bankKeeper.SendCoinsFromModuleToAccount(cacheCtx, types.ModuleName, receiver, fee)
	if err == nil {
		// write the cache
		writeFn()

		// NOTE: The context returned by CacheContext() refers to a new EventManager, so it needs to explicitly set events to the original context.
		ctx.EventManager().EmitEvents(cacheCtx.EventManager().Events())
	}
}

func (k Keeper) RefundFeesOnChannel(ctx sdk.Context, portID, channelID string) error {

	var refundErr error

	k.IteratePacketFeesInEscrow(ctx, portID, channelID, func(packetFees types.PacketFees) (stop bool) {
		for _, identifiedFee := range packetFees.PacketFees {
			refundAccAddr, err := sdk.AccAddressFromBech32(identifiedFee.RefundAddress)
			if err != nil {
				refundErr = err
				return true
			}

			// refund all fees to refund address
			// Use SendCoins rather than the module account send functions since refund address may be a user account or module address.
			// if any `SendCoins` call returns an error, we return error and stop iteration
			if err = k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, refundAccAddr, identifiedFee.Fee.RecvFee); err != nil {
				refundErr = err
				return true
			}
			if err = k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, refundAccAddr, identifiedFee.Fee.AckFee); err != nil {
				refundErr = err
				return true
			}
			if err = k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, refundAccAddr, identifiedFee.Fee.TimeoutFee); err != nil {
				refundErr = err
				return true
			}
		}

		return false
	})

	return refundErr
}
