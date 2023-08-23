package keeper

import (
	"fmt"
	"reflect"

	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v7/internal/collections"
	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	connectiontypes "github.com/cosmos/ibc-go/v7/modules/core/03-connection/types"
	"github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	"github.com/cosmos/ibc-go/v7/modules/core/exported"
)

// ChanUpgradeInit is called by a module to initiate a channel upgrade handshake with
// a module on another chain.
func (k Keeper) ChanUpgradeInit(
	ctx sdk.Context,
	portID string,
	channelID string,
	upgradeFields types.UpgradeFields,
) (types.Upgrade, error) {
	channel, found := k.GetChannel(ctx, portID, channelID)
	if !found {
		return types.Upgrade{}, errorsmod.Wrapf(types.ErrChannelNotFound, "port ID (%s) channel ID (%s)", portID, channelID)
	}

	if channel.State != types.OPEN {
		return types.Upgrade{}, errorsmod.Wrapf(types.ErrInvalidChannelState, "expected %s, got %s", types.OPEN, channel.State)
	}

	if err := k.validateSelfUpgradeFields(ctx, upgradeFields, channel); err != nil {
		return types.Upgrade{}, err
	}

	return types.Upgrade{Fields: upgradeFields}, nil
}

// WriteUpgradeInitChannel writes a channel which has successfully passed the UpgradeInit handshake step.
// An event is emitted for the handshake step.
func (k Keeper) WriteUpgradeInitChannel(ctx sdk.Context, portID, channelID string, upgrade types.Upgrade) types.Channel {
	defer telemetry.IncrCounter(1, "ibc", "channel", "upgrade-init")

	channel, found := k.GetChannel(ctx, portID, channelID)
	if !found {
		panic(fmt.Sprintf("could not find existing channel when updating channel state in successful ChanUpgradeInit step, channelID: %s, portID: %s", channelID, portID))
	}

	channel.UpgradeSequence++

	k.SetChannel(ctx, portID, channelID, channel)
	k.SetUpgrade(ctx, portID, channelID, upgrade)

	k.Logger(ctx).Info("channel state updated", "port-id", portID, "channel-id", channelID, "state", channel.State, "upgrade-sequence", fmt.Sprintf("%d", channel.UpgradeSequence))

	emitChannelUpgradeInitEvent(ctx, portID, channelID, channel, upgrade)
	return channel
}

// ChanUpgradeTry is called by a module to accept the first step of a channel upgrade handshake initiated by
// a module on another chain. If this function is successful, the proposed upgrade will be returned. If the upgrade fails, the upgrade sequence will still be incremented but an error will be returned.
func (k Keeper) ChanUpgradeTry(
	ctx sdk.Context,
	portID,
	channelID string,
	proposedConnectionHops []string,
	counterpartyUpgradeFields types.UpgradeFields,
	counterpartyUpgradeSequence uint64,
	proofCounterpartyChannel,
	proofCounterpartyUpgrade []byte,
	proofHeight clienttypes.Height,
) (types.Upgrade, error) {
	channel, found := k.GetChannel(ctx, portID, channelID)
	if !found {
		return types.Upgrade{}, errorsmod.Wrapf(types.ErrChannelNotFound, "port ID (%s) channel ID (%s)", portID, channelID)
	}

	if !channel.IsOpen() {
		return types.Upgrade{}, errorsmod.Wrapf(types.ErrInvalidChannelState, "expected %s, got %s", types.OPEN, channel.State)
	}

	connection, found := k.connectionKeeper.GetConnection(ctx, channel.ConnectionHops[0])
	if !found {
		return types.Upgrade{}, errorsmod.Wrap(connectiontypes.ErrConnectionNotFound, channel.ConnectionHops[0])
	}

	if connection.GetState() != int32(connectiontypes.OPEN) {
		return types.Upgrade{}, errorsmod.Wrapf(
			connectiontypes.ErrInvalidConnectionState, "connection state is not OPEN (got %s)", connectiontypes.State(connection.GetState()).String(),
		)
	}

	// construct counterpartyChannel from existing information and provided counterpartyUpgradeSequence
	// create upgrade fields from counterparty proposed upgrade and own verified connection hops
	proposedUpgradeFields := types.UpgradeFields{
		Ordering:       counterpartyUpgradeFields.Ordering,
		ConnectionHops: proposedConnectionHops,
		Version:        counterpartyUpgradeFields.Version,
	}

	var (
		err     error
		upgrade types.Upgrade
	)

	// NOTE: if an upgrade exists (crossing hellos) then use existing upgrade fields
	// otherwise, run the upgrade init sub-protocol
	upgrade, found = k.GetUpgrade(ctx, portID, channelID)
	if found {
		proposedUpgradeFields = upgrade.Fields
	} else {
		// if the counterparty sequence is greater than the current sequence, we fast-forward to the counterparty sequence.
		if counterpartyUpgradeSequence > channel.UpgradeSequence {
			// using -1 as WriteUpgradeInitChannel increments the sequence
			channel.UpgradeSequence = counterpartyUpgradeSequence - 1
			k.SetChannel(ctx, portID, channelID, channel)
		}

		// NOTE: OnChanUpgradeInit will not be executed by the application
		upgrade, err = k.ChanUpgradeInit(ctx, portID, channelID, proposedUpgradeFields)
		if err != nil {
			return types.Upgrade{}, errorsmod.Wrap(err, "failed to initialize upgrade")
		}

		channel = k.WriteUpgradeInitChannel(ctx, portID, channelID, upgrade)
	}

	if err := k.checkForUpgradeCompatibility(ctx, proposedUpgradeFields, counterpartyUpgradeFields); err != nil {
		return types.Upgrade{}, errorsmod.Wrap(err, "failed upgrade compatibility check")
	}

	// construct expected counterparty channel from information in state
	// only the counterpartyUpgradeSequence is provided by the relayer
	counterpartyConnectionHops := []string{connection.GetCounterparty().GetConnectionID()}
	counterpartyChannel := types.Channel{
		State:           types.OPEN,
		Ordering:        channel.Ordering,
		Counterparty:    types.NewCounterparty(portID, channelID),
		ConnectionHops:  counterpartyConnectionHops,
		Version:         channel.Version,
		UpgradeSequence: counterpartyUpgradeSequence, // provided by the relayer
		FlushStatus:     types.NOTINFLUSH,
	}

	// verify the counterparty channel state containing the upgrade sequence
	if err := k.connectionKeeper.VerifyChannelState(
		ctx,
		connection,
		proofHeight, proofCounterpartyChannel,
		channel.Counterparty.PortId,
		channel.Counterparty.ChannelId,
		counterpartyChannel,
	); err != nil {
		return types.Upgrade{}, errorsmod.Wrap(err, "failed to verify counterparty channel state")
	}

	if counterpartyUpgradeSequence < channel.UpgradeSequence {
		return upgrade, types.NewUpgradeError(channel.UpgradeSequence-1, errorsmod.Wrapf(
			types.ErrInvalidUpgradeSequence, "counterparty upgrade sequence < current upgrade sequence (%d < %d)", counterpartyUpgradeSequence, channel.UpgradeSequence,
		))
	}

	// verifies the proof that a particular proposed upgrade has been stored in the upgrade path of the counterparty
	if err := k.connectionKeeper.VerifyChannelUpgrade(
		ctx,
		channel.Counterparty.PortId,
		channel.Counterparty.ChannelId,
		connection,
		types.NewUpgrade(counterpartyUpgradeFields, types.Timeout{}, 0),
		proofCounterpartyUpgrade, proofHeight,
	); err != nil {
		return types.Upgrade{}, errorsmod.Wrap(err, "failed to verify counterparty upgrade")
	}

	if err := k.startFlushing(ctx, portID, channelID, &upgrade); err != nil {
		return types.Upgrade{}, err
	}

	return upgrade, nil
}

// WriteUpgradeTryChannel writes the channel end and upgrade to state after successfully passing the UpgradeTry handshake step.
// An event is emitted for the handshake step.
func (k Keeper) WriteUpgradeTryChannel(ctx sdk.Context, portID, channelID string, upgrade types.Upgrade, upgradeVersion string, counterpartyUpgradeFields types.UpgradeFields) (types.Channel, types.Upgrade) {
	defer telemetry.IncrCounter(1, "ibc", "channel", "upgrade-try")

	channel, found := k.GetChannel(ctx, portID, channelID)
	if !found {
		panic(fmt.Sprintf("could not find existing channel when updating channel state in successful ChanUpgradeTry step, channelID: %s, portID: %s", channelID, portID))
	}

	upgrade.Fields.Version = upgradeVersion
	k.SetUpgrade(ctx, portID, channelID, upgrade)

	counterpartyUpgrade := types.Upgrade{Fields: counterpartyUpgradeFields}
	k.SetCounterpartyUpgrade(ctx, portID, channelID, counterpartyUpgrade)

	k.Logger(ctx).Info("channel state updated", "port-id", portID, "channel-id", channelID, "previous-state", types.OPEN, "new-state", channel.State)
	emitChannelUpgradeTryEvent(ctx, portID, channelID, channel, upgrade)

	return channel, upgrade
}

// ChanUpgradeAck is called by a module to accept the ACKUPGRADE handshake step of the channel upgrade protocol.
// This method should only be called by the IBC core msg server.
// This method will verify that the counterparty has called the ChanUpgradeTry handler.
// and that its own upgrade is compatible with the selected counterparty version.
func (k Keeper) ChanUpgradeAck(
	ctx sdk.Context,
	portID,
	channelID string,
	counterpartyUpgrade types.Upgrade,
	proofChannel,
	proofUpgrade []byte,
	proofHeight clienttypes.Height,
) error {
	channel, found := k.GetChannel(ctx, portID, channelID)
	if !found {
		return errorsmod.Wrapf(types.ErrChannelNotFound, "port ID (%s) channel ID (%s)", portID, channelID)
	}

	if !collections.Contains(channel.State, []types.State{types.OPEN, types.STATE_FLUSHING}) {
		return errorsmod.Wrapf(types.ErrInvalidChannelState, "expected one of [%s, %s], got %s", types.OPEN, types.STATE_FLUSHING, channel.State)
	}

	connection, found := k.connectionKeeper.GetConnection(ctx, channel.ConnectionHops[0])
	if !found {
		return errorsmod.Wrap(connectiontypes.ErrConnectionNotFound, channel.ConnectionHops[0])
	}

	if connection.GetState() != int32(connectiontypes.OPEN) {
		return errorsmod.Wrapf(connectiontypes.ErrInvalidConnectionState, "connection state is not OPEN (got %s)", connectiontypes.State(connection.GetState()).String())
	}

	counterpartyHops := []string{connection.GetCounterparty().GetConnectionID()}
	counterpartyChannel := types.Channel{
		State:           types.STATE_FLUSHING,
		Ordering:        channel.Ordering,
		ConnectionHops:  counterpartyHops,
		Counterparty:    types.NewCounterparty(portID, channelID),
		Version:         channel.Version,
		UpgradeSequence: channel.UpgradeSequence,
	}

	// verify the counterparty channel state containing the upgrade sequence
	if err := k.connectionKeeper.VerifyChannelState(
		ctx,
		connection,
		proofHeight, proofChannel,
		channel.Counterparty.PortId,
		channel.Counterparty.ChannelId,
		counterpartyChannel,
	); err != nil {
		return errorsmod.Wrap(err, "failed to verify counterparty channel state")
	}

	// verifies the proof that a particular proposed upgrade has been stored in the upgrade path of the counterparty
	if err := k.connectionKeeper.VerifyChannelUpgrade(
		ctx,
		channel.Counterparty.PortId,
		channel.Counterparty.ChannelId,
		connection,
		counterpartyUpgrade,
		proofUpgrade, proofHeight,
	); err != nil {
		return errorsmod.Wrap(err, "failed to verify counterparty upgrade")
	}

	upgrade, found := k.GetUpgrade(ctx, portID, channelID)
	if !found {
		return errorsmod.Wrapf(types.ErrUpgradeNotFound, "failed to retrieve channel upgrade: port ID (%s) channel ID (%s)", portID, channelID)
	}

	// optimistically accept version that TRY chain proposes and pass this to callback for confirmation
	// in the crossing hello case, we do not modify version that our TRY call returned and instead enforce
	// that both TRY calls returned the same version
	if channel.IsOpen() {
		upgrade.Fields.Version = counterpartyUpgrade.Fields.Version
	}

	// if upgrades are not compatible by ACK step, then we restore the channel
	if err := k.checkForUpgradeCompatibility(ctx, upgrade.Fields, counterpartyUpgrade.Fields); err != nil {
		return types.NewUpgradeError(channel.UpgradeSequence, err)
	}

	if channel.IsOpen() {
		if err := k.startFlushing(ctx, portID, channelID, &upgrade); err != nil {
			return err
		}
	}

	timeout := counterpartyUpgrade.Timeout
	if hasPassed, err := timeout.HasPassed(ctx); hasPassed {
		return types.NewUpgradeError(channel.UpgradeSequence, errorsmod.Wrap(err, "counterparty upgrade timeout has passed"))
	}

	return nil
}

// WriteUpgradeAckChannel writes a channel which has successfully passed the UpgradeAck handshake step as well as
// setting the upgrade for that channel.
// An event is emitted for the handshake step.
func (k Keeper) WriteUpgradeAckChannel(ctx sdk.Context, portID, channelID string, counterpartyUpgrade types.Upgrade) {
	defer telemetry.IncrCounter(1, "ibc", "channel", "upgrade-ack")

	channel, found := k.GetChannel(ctx, portID, channelID)
	if !found {
		panic(fmt.Sprintf("could not find existing channel when updating channel state in successful ChanUpgradeAck step, channelID: %s, portID: %s", channelID, portID))
	}

	if !k.HasInflightPackets(ctx, portID, channelID) {
		channel.State = types.STATE_FLUSHCOMPLETE
	} else {
		k.SetCounterpartyUpgrade(ctx, portID, channelID, counterpartyUpgrade)
	}

	k.SetChannel(ctx, portID, channelID, channel)

	upgrade, found := k.GetUpgrade(ctx, portID, channelID)
	if !found {
		panic(fmt.Sprintf("could not find existing upgrade when updating channel state in successful ChanUpgradeAck step, channelID: %s, portID: %s", channelID, portID))
	}

	upgrade.Fields.Version = counterpartyUpgrade.Fields.Version

	k.SetUpgrade(ctx, portID, channelID, upgrade)

	k.Logger(ctx).Info("channel state updated", "port-id", portID, "channel-id", channelID, "state", channel.State.String())
	emitChannelUpgradeAckEvent(ctx, portID, channelID, channel, upgrade)
}

// ChanUpgradeConfirm is called on the chain which is on FLUSHING after chanUpgradeAck is called on the counterparty.
// This will inform the TRY chain of the timeout set on ACK by the counterparty. If the timeout has already exceeded, we will write an error receipt and restore.
func (k Keeper) ChanUpgradeConfirm(
	ctx sdk.Context,
	portID,
	channelID string,
	counterpartyChannelState types.State,
	counterpartyUpgrade types.Upgrade,
	proofChannel,
	proofUpgrade []byte,
	proofHeight clienttypes.Height,
) error {
	channel, found := k.GetChannel(ctx, portID, channelID)
	if !found {
		return errorsmod.Wrapf(types.ErrChannelNotFound, "port ID (%s) channel ID (%s)", portID, channelID)
	}

	if channel.State != types.STATE_FLUSHING {
		return errorsmod.Wrapf(types.ErrInvalidChannelState, "expected %s, got %s", types.STATE_FLUSHING, channel.State)
	}

	if !collections.Contains(counterpartyChannelState, []types.State{types.STATE_FLUSHING, types.STATE_FLUSHCOMPLETE}) {
		return errorsmod.Wrapf(types.ErrInvalidCounterparty, "expected one of [%s, %s], got %s", types.STATE_FLUSHING, types.STATE_FLUSHCOMPLETE, counterpartyChannelState)
	}

	connection, found := k.connectionKeeper.GetConnection(ctx, channel.ConnectionHops[0])
	if !found {
		return errorsmod.Wrap(connectiontypes.ErrConnectionNotFound, channel.ConnectionHops[0])
	}

	if connection.GetState() != int32(connectiontypes.OPEN) {
		return errorsmod.Wrapf(connectiontypes.ErrInvalidConnectionState, "connection state is not OPEN (got %s)", connectiontypes.State(connection.GetState()).String())
	}

	counterpartyHops := []string{connection.GetCounterparty().GetConnectionID()}
	counterpartyChannel := types.Channel{
		State:           counterpartyChannelState,
		Ordering:        channel.Ordering,
		ConnectionHops:  counterpartyHops,
		Counterparty:    types.NewCounterparty(portID, channelID),
		Version:         channel.Version,
		UpgradeSequence: channel.UpgradeSequence,
	}

	if err := k.connectionKeeper.VerifyChannelState(
		ctx,
		connection,
		proofHeight, proofChannel,
		channel.Counterparty.PortId,
		channel.Counterparty.ChannelId,
		counterpartyChannel,
	); err != nil {
		return errorsmod.Wrap(err, "failed to verify counterparty channel state")
	}

	if err := k.connectionKeeper.VerifyChannelUpgrade(
		ctx,
		channel.Counterparty.PortId,
		channel.Counterparty.ChannelId,
		connection,
		counterpartyUpgrade,
		proofUpgrade, proofHeight,
	); err != nil {
		return errorsmod.Wrap(err, "failed to verify counterparty upgrade")
	}

	timeout := counterpartyUpgrade.Timeout
	if hasPassed, err := timeout.HasPassed(ctx); hasPassed {
		return types.NewUpgradeError(channel.UpgradeSequence, errorsmod.Wrap(err, "counterparty upgrade timeout has passed"))
	}

	return nil
}

// WriteUpgradeConfirmChannel writes a channel which has successfully passed the ChanUpgradeConfirm handshake step.
// If the channel has no in-flight packets, its state is updated to indicate that flushing has completed. Otherwise, the counterparty upgrade is set
// and the channel state is left unchanged.
// An event is emitted for the handshake step.
func (k Keeper) WriteUpgradeConfirmChannel(ctx sdk.Context, portID, channelID string, counterpartyUpgrade types.Upgrade) {
	defer telemetry.IncrCounter(1, "ibc", "channel", "upgrade-confirm")

	channel, found := k.GetChannel(ctx, portID, channelID)
	if !found {
		panic(fmt.Sprintf("could not find existing channel when updating channel state in successful ChanUpgradeConfirm step, channelID: %s, portID: %s", channelID, portID))
	}

	if !k.HasInflightPackets(ctx, portID, channelID) {
		previousState := channel.State
		channel.State = types.STATE_FLUSHCOMPLETE
		k.SetChannel(ctx, portID, channelID, channel)

		k.Logger(ctx).Info("channel state updated", "port-id", portID, "channel-id", channelID, "previous-state", previousState, "new-state", channel.State)
	} else {
		k.SetCounterpartyUpgrade(ctx, portID, channelID, counterpartyUpgrade)
	}

	emitChannelUpgradeConfirmEvent(ctx, portID, channelID, channel)
}

// ChanUpgradeOpen is called by a module to complete the channel upgrade handshake and move the channel back to an OPEN state.
// This method should only be called after both channels have flushed any in-flight packets.
// This method should only be called directly by the core IBC message server.
func (k Keeper) ChanUpgradeOpen(
	ctx sdk.Context,
	portID,
	channelID string,
	counterpartyChannelState types.State,
	proofCounterpartyChannel []byte,
	proofHeight clienttypes.Height,
) error {
	channel, found := k.GetChannel(ctx, portID, channelID)
	if !found {
		return errorsmod.Wrapf(types.ErrChannelNotFound, "port ID (%s) channel ID (%s)", portID, channelID)
	}

	if channel.State != types.STATE_FLUSHCOMPLETE {
		return errorsmod.Wrapf(types.ErrInvalidChannelState, "expected %s, got %s", types.STATE_FLUSHCOMPLETE, channel.State)
	}

	connection, found := k.connectionKeeper.GetConnection(ctx, channel.ConnectionHops[0])
	if !found {
		return errorsmod.Wrap(connectiontypes.ErrConnectionNotFound, channel.ConnectionHops[0])
	}

	if connection.GetState() != int32(connectiontypes.OPEN) {
		return errorsmod.Wrapf(connectiontypes.ErrInvalidConnectionState, "connection state is not OPEN (got %s)", connectiontypes.State(connection.GetState()).String())
	}

	var counterpartyChannel types.Channel
	switch counterpartyChannelState {
	case types.OPEN:
		upgrade, found := k.GetUpgrade(ctx, portID, channelID)
		if !found {
			return errorsmod.Wrapf(types.ErrUpgradeNotFound, "failed to retrieve channel upgrade: port ID (%s) channel ID (%s)", portID, channelID)
		}
		// If counterparty has reached OPEN, we must use the upgraded connection to verify the counterparty channel
		upgradeConnection, found := k.connectionKeeper.GetConnection(ctx, upgrade.Fields.ConnectionHops[0])
		if !found {
			return errorsmod.Wrap(connectiontypes.ErrConnectionNotFound, upgrade.Fields.ConnectionHops[0])
		}

		if upgradeConnection.GetState() != int32(connectiontypes.OPEN) {
			return errorsmod.Wrapf(connectiontypes.ErrInvalidConnectionState, "connection state is not OPEN (got %s)", connectiontypes.State(upgradeConnection.GetState()).String())
		}

		counterpartyChannel = types.Channel{
			State:           types.OPEN,
			Ordering:        upgrade.Fields.Ordering,
			ConnectionHops:  []string{upgradeConnection.GetCounterparty().GetConnectionID()},
			Counterparty:    types.NewCounterparty(portID, channelID),
			Version:         upgrade.Fields.Version,
			UpgradeSequence: channel.UpgradeSequence,
		}

	case types.STATE_FLUSHCOMPLETE:
		counterpartyChannel = types.Channel{
			State:           types.STATE_FLUSHCOMPLETE,
			Ordering:        channel.Ordering,
			ConnectionHops:  []string{connection.GetCounterparty().GetConnectionID()},
			Counterparty:    types.NewCounterparty(portID, channelID),
			Version:         channel.Version,
			UpgradeSequence: channel.UpgradeSequence,
		}

	default:
		return errorsmod.Wrapf(types.ErrInvalidCounterparty, "counterparty channel state must be one of [%s, %s], got %s", types.OPEN, types.STATE_FLUSHCOMPLETE, counterpartyChannelState)
	}

	if err := k.connectionKeeper.VerifyChannelState(
		ctx,
		connection,
		proofHeight, proofCounterpartyChannel,
		channel.Counterparty.PortId,
		channel.Counterparty.ChannelId,
		counterpartyChannel,
	); err != nil {
		return errorsmod.Wrap(err, "failed to verify counterparty channel")
	}

	return nil
}

// WriteUpgradeOpenChannel writes the agreed upon upgrade fields to the channel, and sets the channel state back to OPEN. This can be called in one of two cases:
// - In the UpgradeConfirm step of the handshake if both sides have already flushed all in-flight packets.
// - In the UpgradeOpen step of the handshake.
func (k Keeper) WriteUpgradeOpenChannel(ctx sdk.Context, portID, channelID string) {
	channel, found := k.GetChannel(ctx, portID, channelID)
	if !found {
		panic(fmt.Sprintf("could not find existing channel when updating channel state, channelID: %s, portID: %s", channelID, portID))
	}

	upgrade, found := k.GetUpgrade(ctx, portID, channelID)
	if !found {
		panic(fmt.Sprintf("could not find upgrade when updating channel state, channelID: %s, portID: %s", channelID, portID))
	}

	// Switch channel fields to upgrade fields and set channel state to OPEN
	previousState := channel.State
	channel.Ordering = upgrade.Fields.Ordering
	channel.Version = upgrade.Fields.Version
	channel.ConnectionHops = upgrade.Fields.ConnectionHops
	channel.State = types.OPEN

	k.SetChannel(ctx, portID, channelID, channel)

	// delete state associated with upgrade which is no longer required.
	k.deleteUpgradeInfo(ctx, portID, channelID)

	k.Logger(ctx).Info("channel state updated", "port-id", portID, "channel-id", channelID, "previous-state", previousState.String(), "new-state", types.OPEN.String())
	emitChannelUpgradeOpenEvent(ctx, portID, channelID, channel)
}

// ChanUpgradeCancel is called by a module to cancel a channel upgrade that is in progress.
func (k Keeper) ChanUpgradeCancel(ctx sdk.Context, portID, channelID string, errorReceipt types.ErrorReceipt, errorReceiptProof []byte, proofHeight clienttypes.Height, signer string) error {
	channel, found := k.GetChannel(ctx, portID, channelID)
	if !found {
		return errorsmod.Wrapf(types.ErrChannelNotFound, "port ID (%s) channel ID (%s)", portID, channelID)
	}

	_, found = k.GetUpgrade(ctx, portID, channelID)
	if !found {
		return errorsmod.Wrapf(types.ErrUpgradeNotFound, "port ID (%s) channel ID (%s)", portID, channelID)
	}

	if errorReceipt.Sequence < channel.UpgradeSequence {
		return errorsmod.Wrapf(types.ErrInvalidUpgradeSequence, "error receipt sequence (%d) must be greater than or equal to current upgrade sequence (%d)", errorReceipt.Sequence, channel.UpgradeSequence)
	}

	// if the msgSender is authorized to make and cancel upgrades AND the current channel has not already reached FLUSHCOMPLETE
	// then we can restore immediately without any additional checks
	// otherwise, we can only cancel if the counterparty wrote an error receipt during the upgrade handshake
	if k.isAuthorizedUpgrader(signer) && channel.State != types.STATE_FLUSHCOMPLETE {
		return nil
	}

	// get underlying connection for proof verification
	connection, found := k.connectionKeeper.GetConnection(ctx, channel.ConnectionHops[0])
	if !found {
		return errorsmod.Wrap(connectiontypes.ErrConnectionNotFound, channel.ConnectionHops[0])
	}

	if connection.GetState() != int32(connectiontypes.OPEN) {
		return errorsmod.Wrapf(
			connectiontypes.ErrInvalidConnectionState,
			"connection state is not OPEN (got %s)", connectiontypes.State(connection.GetState()).String(),
		)
	}

	if err := k.connectionKeeper.VerifyChannelUpgradeError(
		ctx,
		channel.Counterparty.PortId,
		channel.Counterparty.ChannelId,
		connection,
		errorReceipt,
		errorReceiptProof,
		proofHeight,
	); err != nil {
		return errorsmod.Wrap(err, "failed to verify counterparty error receipt")
	}

	return nil
}

// isAuthorizedUpgrader checks if the sender is authorized to cancel the upgrade.
func (Keeper) isAuthorizedUpgrader(signer string) bool {
	// TODO: the authority is only available on the core ibc keeper at the moment.
	// return k.GetAuthority() == signer
	return true
}

// WriteUpgradeCancelChannel writes a channel which has canceled the upgrade process.Auxiliary upgrade state is
// also deleted.
func (k Keeper) WriteUpgradeCancelChannel(ctx sdk.Context, portID, channelID string, errorReceipt types.ErrorReceipt) {
	defer telemetry.IncrCounter(1, "ibc", "channel", "upgrade-cancel")

	channel, found := k.GetChannel(ctx, portID, channelID)
	if !found {
		panic(fmt.Sprintf("could not find existing channel when updating channel state, channelID: %s, portID: %s", channelID, portID))
	}

	upgrade, found := k.GetUpgrade(ctx, portID, channelID)
	if !found {
		panic(fmt.Sprintf("could not find upgrade when updating channel state, channelID: %s, portID: %s", channelID, portID))
	}

	previousState := channel.State

	k.SetUpgradeErrorReceipt(ctx, portID, channelID, errorReceipt)
	channel = k.restoreChannel(ctx, portID, channelID, errorReceipt.Sequence, channel)

	k.Logger(ctx).Info("channel state updated", "port-id", portID, "channel-id", channelID, "previous-state", previousState, "new-state", types.OPEN.String())
	emitChannelUpgradeCancelEvent(ctx, portID, channelID, channel, upgrade)
}

// ChanUpgradeTimeout times out an outstanding upgrade.
// This should be used by the initialising chain when the counterparty chain has not responded to an upgrade proposal within the specified timeout period.
func (k Keeper) ChanUpgradeTimeout(
	ctx sdk.Context,
	portID, channelID string,
	counterpartyChannel types.Channel,
	prevErrorReceipt *types.ErrorReceipt,
	proofCounterpartyChannel,
	proofErrorReceipt []byte,
	proofHeight exported.Height,
) error {
	channel, found := k.GetChannel(ctx, portID, channelID)
	if !found {
		return errorsmod.Wrapf(types.ErrChannelNotFound, "port ID (%s) channel ID (%s)", portID, channelID)
	}

	if channel.State != types.INITUPGRADE {
		return errorsmod.Wrapf(types.ErrInvalidChannelState, "channel state is not INITUPGRADE (got %s)", channel.State)
	}

	upgrade, found := k.GetUpgrade(ctx, portID, channelID)
	if !found {
		return errorsmod.Wrapf(types.ErrUpgradeNotFound, "port ID (%s) channel ID (%s)", portID, channelID)
	}

	connection, found := k.connectionKeeper.GetConnection(ctx, channel.ConnectionHops[0])
	if !found {
		return errorsmod.Wrap(
			connectiontypes.ErrConnectionNotFound,
			channel.ConnectionHops[0],
		)
	}

	if connection.GetState() != int32(connectiontypes.OPEN) {
		return errorsmod.Wrapf(
			connectiontypes.ErrInvalidConnectionState,
			"connection state is not OPEN (got %s)", connectiontypes.State(connection.GetState()).String(),
		)
	}

	// proof must be from a height after timeout has elapsed. Either timeoutHeight or timeoutTimestamp must be defined.
	// if timeoutHeight is defined and proof is from before timeout height, abort transaction
	proofTimestamp, err := k.connectionKeeper.GetTimestampAtHeight(ctx, connection, proofHeight)
	if err != nil {
		return err
	}

	timeout := upgrade.Timeout
	proofHeightIsInvalid := timeout.Height.IsZero() || proofHeight.LT(timeout.Height)
	proofTimestampIsInvalid := timeout.Timestamp == 0 || proofTimestamp < timeout.Timestamp
	if proofHeightIsInvalid && proofTimestampIsInvalid {
		return errorsmod.Wrap(types.ErrInvalidUpgradeTimeout, "timeout has not yet passed on counterparty chain")
	}

	// counterparty channel must be proved to still be in OPEN state or INITUPGRADE state (crossing hellos)
	if !collections.Contains(counterpartyChannel.State, []types.State{types.OPEN, types.INITUPGRADE}) {
		return errorsmod.Wrapf(types.ErrInvalidChannelState, "expected one of [%s, %s], got %s", types.OPEN, types.INITUPGRADE, counterpartyChannel.State)
	}

	// verify the counterparty channel state
	if err := k.connectionKeeper.VerifyChannelState(
		ctx,
		connection,
		proofHeight, proofCounterpartyChannel,
		channel.Counterparty.PortId,
		channel.Counterparty.ChannelId,
		counterpartyChannel,
	); err != nil {
		return errorsmod.Wrap(err, "failed to verify counterparty channel state")
	}

	// Error receipt passed in is either nil or it is a stale error receipt from a previous upgrade
	if prevErrorReceipt == nil {
		if err := k.connectionKeeper.VerifyChannelUpgradeErrorAbsence(
			ctx,
			channel.Counterparty.PortId, channel.Counterparty.ChannelId,
			connection,
			proofErrorReceipt,
			proofHeight,
		); err != nil {
			return errorsmod.Wrap(err, "failed to verify absence of counterparty channel upgrade error receipt")
		}

		return nil
	}
	// timeout for this sequence can only succeed if the error receipt written into the error path on the counterparty
	// was for a previous sequence by the timeout deadline.
	upgradeSequence := channel.UpgradeSequence
	if upgradeSequence <= prevErrorReceipt.Sequence {
		return errorsmod.Wrapf(types.ErrInvalidUpgradeSequence, "previous counterparty error receipt sequence is greater than or equal to our current upgrade sequence: %d > %d", prevErrorReceipt.Sequence, upgradeSequence)
	}

	if err := k.connectionKeeper.VerifyChannelUpgradeError(
		ctx,
		channel.Counterparty.PortId, channel.Counterparty.ChannelId,
		connection,
		*prevErrorReceipt,
		proofErrorReceipt,
		proofHeight,
	); err != nil {
		return errorsmod.Wrap(err, "failed to verify counterparty channel upgrade error receipt")
	}

	return nil
}

// WriteUpgradeTimeoutChannel restores the channel state of an initialising chain in the event that the counterparty chain has passed the timeout set in ChanUpgradeInit to the state before the upgrade was proposed.
// Auxiliary upgrade state is also deleted.
// An event is emitted for the handshake step.
func (k Keeper) WriteUpgradeTimeoutChannel(
	ctx sdk.Context,
	portID, channelID string,
) {
	defer telemetry.IncrCounter(1, "ibc", "channel", "upgrade-timeout")

	channel, found := k.GetChannel(ctx, portID, channelID)
	if !found {
		panic(fmt.Sprintf("could not find existing channel when updating channel state in successful ChanUpgradeTimeout step, channelID: %s, portID: %s", channelID, portID))
	}

	upgrade, found := k.GetUpgrade(ctx, portID, channelID)
	if !found {
		panic(fmt.Sprintf("could not find existing upgrade when cancelling channel upgrade, channelID: %s, portID: %s", channelID, portID))
	}

	channel = k.restoreChannel(ctx, portID, channelID, channel.UpgradeSequence, channel)

	k.Logger(ctx).Info("channel state restored", "port-id", portID, "channel-id", channelID)
	emitChannelUpgradeTimeoutEvent(ctx, portID, channelID, channel, upgrade)
}

// startFlushing will set the upgrade last packet send and continue blocking the upgrade from continuing until all
// in-flight packets have been flushed.
func (k Keeper) startFlushing(ctx sdk.Context, portID, channelID string, upgrade *types.Upgrade) error {
	channel, found := k.GetChannel(ctx, portID, channelID)
	if !found {
		return errorsmod.Wrapf(types.ErrChannelNotFound, "port ID (%s) channel ID (%s)", portID, channelID)
	}

	connection, found := k.connectionKeeper.GetConnection(ctx, channel.ConnectionHops[0])
	if !found {
		return errorsmod.Wrap(connectiontypes.ErrConnectionNotFound, channel.ConnectionHops[0])
	}

	if connection.GetState() != int32(connectiontypes.OPEN) {
		return errorsmod.Wrapf(connectiontypes.ErrInvalidConnectionState, "connection state is not OPEN (got %s)", connectiontypes.State(connection.GetState()).String())
	}

	channel.State = types.STATE_FLUSHING
	k.SetChannel(ctx, portID, channelID, channel)

	nextSequenceSend, found := k.GetNextSequenceSend(ctx, portID, channelID)
	if !found {
		return errorsmod.Wrapf(types.ErrSequenceSendNotFound, "port ID (%s) channel ID (%s)", portID, channelID)
	}

	upgrade.LatestSequenceSend = nextSequenceSend - 1
	upgrade.Timeout = k.getUpgradeTimeout(ctx)
	k.SetUpgrade(ctx, portID, channelID, *upgrade)

	return nil
}

// getUpgradeTimeout returns the timeout for the given upgrade.
func (k Keeper) getUpgradeTimeout(ctx sdk.Context) types.Timeout {
	return k.GetParams(ctx).UpgradeTimeout
}

// syncUpgradeSequence ensures current upgrade handshake only continues if both channels are using the same upgrade sequence,
// otherwise an upgrade error is returned so that an error receipt will be written so that the upgrade handshake may be attempted again with synchronized sequences.
func (k Keeper) syncUpgradeSequence(ctx sdk.Context, portID, channelID string, channel types.Channel, counterpartyUpgradeSequence uint64) error {
	// save the previous upgrade sequence for the error message
	prevUpgradeSequence := channel.UpgradeSequence

	if counterpartyUpgradeSequence != channel.UpgradeSequence {
		// error on the higher sequence so that both chains synchronize on a fresh sequence
		channel.UpgradeSequence = sdkmath.Max(counterpartyUpgradeSequence, channel.UpgradeSequence)
		k.SetChannel(ctx, portID, channelID, channel)

		return types.NewUpgradeError(channel.UpgradeSequence, errorsmod.Wrapf(
			types.ErrInvalidUpgradeSequence, "expected upgrade sequence (%d) to match counterparty upgrade sequence (%d)", prevUpgradeSequence, counterpartyUpgradeSequence),
		)
	}

	return nil
}

// checkForUpgradeCompatibility checks performs stateful validation of self upgrade fields relative to counterparty upgrade.
func (k Keeper) checkForUpgradeCompatibility(ctx sdk.Context, upgradeFields, counterpartyUpgradeFields types.UpgradeFields) error {
	// assert that both sides propose the same channel ordering
	if upgradeFields.Ordering != counterpartyUpgradeFields.Ordering {
		return errorsmod.Wrapf(types.ErrIncompatibleCounterpartyUpgrade, "expected upgrade ordering (%s) to match counterparty upgrade ordering (%s)", upgradeFields.Ordering, counterpartyUpgradeFields.Ordering)
	}

	if upgradeFields.Version != counterpartyUpgradeFields.Version {
		return errorsmod.Wrapf(types.ErrIncompatibleCounterpartyUpgrade, "expected upgrade version (%s) to match counterparty upgrade version (%s)", upgradeFields.Version, counterpartyUpgradeFields.Version)
	}

	connection, found := k.connectionKeeper.GetConnection(ctx, upgradeFields.ConnectionHops[0])
	if !found {
		// NOTE: this error is expected to be unreachable as the proposed upgrade connectionID should have been
		// validated in the upgrade INIT and TRY handlers
		return errorsmod.Wrap(connectiontypes.ErrConnectionNotFound, upgradeFields.ConnectionHops[0])
	}

	if connection.GetState() != int32(connectiontypes.OPEN) {
		// NOTE: this error is expected to be unreachable as the proposed upgrade connectionID should have been
		// validated in the upgrade INIT and TRY handlers
		return errorsmod.Wrapf(connectiontypes.ErrInvalidConnectionState, "expected proposed connection to be OPEN (got %s)", connectiontypes.State(connection.GetState()).String())
	}

	// connectionHops can change in a channelUpgrade, however both sides must still be each other's counterparty.
	if counterpartyUpgradeFields.ConnectionHops[0] != connection.GetCounterparty().GetConnectionID() {
		return errorsmod.Wrapf(
			types.ErrIncompatibleCounterpartyUpgrade, "counterparty upgrade connection end is not a counterparty of self proposed connection end (%s != %s)", counterpartyUpgradeFields.ConnectionHops[0], connection.GetCounterparty().GetConnectionID())
	}

	return nil
}

// validateSelfUpgradeFields validates the proposed upgrade fields against the existing channel.
// It returns an error if the following constraints are not met:
// - there exists at least one valid proposed change to the existing channel fields
// - the proposed order is a subset of the existing order
// - the proposed connection hops do not exist
// - the proposed version is non-empty (checked in UpgradeFields.ValidateBasic())
// - the proposed connection hops are not open
func (k Keeper) validateSelfUpgradeFields(ctx sdk.Context, proposedUpgrade types.UpgradeFields, currentChannel types.Channel) error {
	currentFields := extractUpgradeFields(currentChannel)

	if reflect.DeepEqual(proposedUpgrade, currentFields) {
		return errorsmod.Wrapf(types.ErrChannelExists, "existing channel end is identical to proposed upgrade channel end: got %s", proposedUpgrade)
	}

	connectionID := proposedUpgrade.ConnectionHops[0]
	connection, found := k.connectionKeeper.GetConnection(ctx, connectionID)
	if !found {
		return errorsmod.Wrapf(connectiontypes.ErrConnectionNotFound, "failed to retrieve connection: %s", connectionID)
	}

	if connection.GetState() != int32(connectiontypes.OPEN) {
		return errorsmod.Wrapf(
			connectiontypes.ErrInvalidConnectionState,
			"connection state is not OPEN (got %s)", connectiontypes.State(connection.GetState()).String(),
		)
	}

	getVersions := connection.GetVersions()
	if len(getVersions) != 1 {
		return errorsmod.Wrapf(
			connectiontypes.ErrInvalidVersion,
			"single version must be negotiated on connection before opening channel, got: %v",
			getVersions,
		)
	}

	if !connectiontypes.VerifySupportedFeature(getVersions[0], proposedUpgrade.Ordering.String()) {
		return errorsmod.Wrapf(
			connectiontypes.ErrInvalidVersion,
			"connection version %s does not support channel ordering: %s",
			getVersions[0], proposedUpgrade.Ordering.String(),
		)
	}

	return nil
}

// extractUpgradeFields returns the upgrade fields from the provided channel.
func extractUpgradeFields(channel types.Channel) types.UpgradeFields {
	return types.UpgradeFields{
		Ordering:       channel.Ordering,
		ConnectionHops: channel.ConnectionHops,
		Version:        channel.Version,
	}
}

// MustAbortUpgrade will restore the channel state and flush status to their pre-upgrade state so that upgrade is aborted.
// Any unnecessary state is deleted and an error receipt is written.
// This function is expected to always succeed, a panic will occur if an error occurs.
func (k Keeper) MustAbortUpgrade(ctx sdk.Context, portID, channelID string, err error) {
	if err := k.abortUpgrade(ctx, portID, channelID, err); err != nil {
		panic(err)
	}
}

// abortUpgrade will restore the channel state and flush status to their pre-upgrade state so that upgrade is aborted.
// Any unnecessary state is delete and an error receipt is written.
func (k Keeper) abortUpgrade(ctx sdk.Context, portID, channelID string, err error) error {
	if err == nil {
		return errorsmod.Wrap(types.ErrInvalidUpgradeError, "cannot abort upgrade handshake with nil error")
	}

	upgrade, found := k.GetUpgrade(ctx, portID, channelID)
	if !found {
		return errorsmod.Wrapf(types.ErrUpgradeNotFound, "port ID (%s) channel ID (%s)", portID, channelID)
	}

	channel, found := k.GetChannel(ctx, portID, channelID)
	if !found {
		return errorsmod.Wrapf(types.ErrChannelNotFound, "port ID (%s) channel ID (%s)", portID, channelID)
	}

	// the channel upgrade sequence has already been updated in ChannelUpgradeTry, so we can pass
	// its updated value.
	k.restoreChannel(ctx, portID, channelID, channel.UpgradeSequence, channel)

	// in the case of application callbacks, the error may not be an upgrade error.
	// in this case we need to construct one in order to write the error receipt.
	upgradeError, ok := err.(*types.UpgradeError)
	if !ok {
		upgradeError = types.NewUpgradeError(channel.UpgradeSequence, err)
	}

	if err := k.WriteErrorReceipt(ctx, portID, channelID, upgrade.Fields, upgradeError); err != nil {
		return err
	}

	return nil
}

// restoreChannel will restore the channel state and flush status to their pre-upgrade state so that upgrade is aborted.
func (k Keeper) restoreChannel(ctx sdk.Context, portID, channelID string, upgradeSequence uint64, channel types.Channel) types.Channel {
	channel.State = types.OPEN
	channel.UpgradeSequence = upgradeSequence

	k.SetChannel(ctx, portID, channelID, channel)

	// delete state associated with upgrade which is no longer required.
	k.deleteUpgradeInfo(ctx, portID, channelID)
	return channel
}

// WriteErrorReceipt will write an error receipt from the provided UpgradeError.
func (k Keeper) WriteErrorReceipt(ctx sdk.Context, portID, channelID string, upgradeFields types.UpgradeFields, upgradeError *types.UpgradeError) error {
	channel, found := k.GetChannel(ctx, portID, channelID)
	if !found {
		return errorsmod.Wrapf(types.ErrChannelNotFound, "port ID (%s) channel ID (%s)", portID, channelID)
	}

	k.SetUpgradeErrorReceipt(ctx, portID, channelID, upgradeError.GetErrorReceipt())
	emitErrorReceiptEvent(ctx, portID, channelID, channel, upgradeFields, upgradeError)
	return nil
}
