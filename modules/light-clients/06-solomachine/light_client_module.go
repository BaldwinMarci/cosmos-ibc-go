package solomachine

import (
	errorsmod "cosmossdk.io/errors"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/api"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"
)

var _ api.LightClientModule = (*LightClientModule)(nil)

// LightClientModule implements the core IBC api.LightClientModule interface?
type LightClientModule struct {
	cdc           codec.BinaryCodec
	storeProvider api.ClientStoreProvider
}

// NewLightClientModule creates and returns a new 06-solomachine LightClientModule.
func NewLightClientModule(cdc codec.BinaryCodec) LightClientModule {
	return LightClientModule{
		cdc: cdc,
	}
}

func (l *LightClientModule) RegisterStoreProvider(storeProvider api.ClientStoreProvider) {
	l.storeProvider = storeProvider
}

// Initialize is called upon client creation, it allows the client to perform validation on the initial consensus state and set the
// client state, consensus state and any client-specific metadata necessary for correct light client operation in the provided client store.
func (l LightClientModule) Initialize(ctx sdk.Context, clientID string, clientStateBz, consensusStateBz []byte) error {
	if err := validateClientID(clientID); err != nil {
		return err
	}

	var clientState ClientState
	if err := l.cdc.Unmarshal(clientStateBz, &clientState); err != nil {
		return err
	}

	if err := clientState.Validate(); err != nil {
		return err
	}

	var consensusState ConsensusState
	if err := l.cdc.Unmarshal(consensusStateBz, &consensusState); err != nil {
		return err
	}

	if err := consensusState.ValidateBasic(); err != nil {
		return err
	}

	clientStore := l.storeProvider.ClientStore(ctx, clientID)
	return clientState.Initialize(ctx, l.cdc, clientStore, &consensusState)
}

// VerifyClientMessage must verify a ClientMessage. A ClientMessage could be a Header, Misbehaviour, or batch update.
// It must handle each type of ClientMessage appropriately. Calls to CheckForMisbehaviour, UpdateState, and UpdateStateOnMisbehaviour
// will assume that the content of the ClientMessage has been verified and can be trusted. An error should be returned
// if the ClientMessage fails to verify.
func (l LightClientModule) VerifyClientMessage(ctx sdk.Context, clientID string, clientMsg api.ClientMessage) error {
	if err := validateClientID(clientID); err != nil {
		return err
	}

	clientStore := l.storeProvider.ClientStore(ctx, clientID)
	clientState, found := getClientState(clientStore, l.cdc)
	if !found {
		return errorsmod.Wrap(clienttypes.ErrClientNotFound, clientID)
	}

	return clientState.VerifyClientMessage(ctx, l.cdc, clientStore, clientMsg)
}

// Checks for evidence of a misbehaviour in Header or Misbehaviour type. It assumes the ClientMessage
// has already been verified.
func (l LightClientModule) CheckForMisbehaviour(ctx sdk.Context, clientID string, clientMsg api.ClientMessage) bool {
	if err := validateClientID(clientID); err != nil {
		panic(err)
	}

	clientStore := l.storeProvider.ClientStore(ctx, clientID)
	clientState, found := getClientState(clientStore, l.cdc)
	if !found {
		panic(errorsmod.Wrap(clienttypes.ErrClientNotFound, clientID))
	}

	return clientState.CheckForMisbehaviour(ctx, l.cdc, clientStore, clientMsg)
}

// UpdateStateOnMisbehaviour should perform appropriate state changes on a client state given that misbehaviour has been detected and verified
func (l LightClientModule) UpdateStateOnMisbehaviour(ctx sdk.Context, clientID string, clientMsg api.ClientMessage) {
	if err := validateClientID(clientID); err != nil {
		panic(err)
	}

	clientStore := l.storeProvider.ClientStore(ctx, clientID)
	clientState, found := getClientState(clientStore, l.cdc)
	if !found {
		panic(errorsmod.Wrap(clienttypes.ErrClientNotFound, clientID))
	}

	clientState.UpdateStateOnMisbehaviour(ctx, l.cdc, clientStore, clientMsg)
}

// UpdateState updates and stores as necessary any associated information for an IBC client, such as the ClientState and corresponding ConsensusState.
// Upon successful update, a list of consensus heights is returned. It assumes the ClientMessage has already been verified.
func (l LightClientModule) UpdateState(ctx sdk.Context, clientID string, clientMsg api.ClientMessage) []api.Height {
	if err := validateClientID(clientID); err != nil {
		panic(err)
	}

	clientStore := l.storeProvider.ClientStore(ctx, clientID)
	clientState, found := getClientState(clientStore, l.cdc)
	if !found {
		panic(errorsmod.Wrap(clienttypes.ErrClientNotFound, clientID))
	}

	return clientState.UpdateState(ctx, l.cdc, clientStore, clientMsg)
}

// VerifyMembership is a generic proof verification method which verifies a proof of the existence of a value at a given CommitmentPath at the specified height.
// The caller is expected to construct the full CommitmentPath from a CommitmentPrefix and a standardized path (as defined in ICS 24).
func (l LightClientModule) VerifyMembership(
	ctx sdk.Context,
	clientID string,
	height api.Height, // TODO: change to concrete type
	delayTimePeriod uint64,
	delayBlockPeriod uint64,
	proof []byte,
	path api.Path, // TODO: change to conrete type
	value []byte,
) error {
	if err := validateClientID(clientID); err != nil {
		return err
	}

	clientStore := l.storeProvider.ClientStore(ctx, clientID)
	clientState, found := getClientState(clientStore, l.cdc)
	if !found {
		return errorsmod.Wrap(clienttypes.ErrClientNotFound, clientID)
	}

	return clientState.VerifyMembership(ctx, clientStore, l.cdc, height, delayTimePeriod, delayBlockPeriod, proof, path, value)
}

// VerifyNonMembership is a generic proof verification method which verifies the absence of a given CommitmentPath at a specified height.
// The caller is expected to construct the full CommitmentPath from a CommitmentPrefix and a standardized path (as defined in ICS 24).
func (l LightClientModule) VerifyNonMembership(
	ctx sdk.Context,
	clientID string,
	height api.Height, // TODO: change to concrete type
	delayTimePeriod uint64,
	delayBlockPeriod uint64,
	proof []byte,
	path api.Path, // TODO: change to conrete type
) error {
	if err := validateClientID(clientID); err != nil {
		return err
	}

	clientStore := l.storeProvider.ClientStore(ctx, clientID)
	clientState, found := getClientState(clientStore, l.cdc)
	if !found {
		return errorsmod.Wrap(clienttypes.ErrClientNotFound, clientID)
	}

	return clientState.VerifyNonMembership(ctx, clientStore, l.cdc, height, delayTimePeriod, delayBlockPeriod, proof, path)
}

// Status must return the status of the client. Only Active clients are allowed to process packets.
func (l LightClientModule) Status(ctx sdk.Context, clientID string) api.Status {
	if err := validateClientID(clientID); err != nil {
		return api.Unknown // TODO: or panic
	}

	clientStore := l.storeProvider.ClientStore(ctx, clientID)
	clientState, found := getClientState(clientStore, l.cdc)
	if !found {
		return api.Unknown
	}

	return clientState.Status(ctx, clientStore, l.cdc)
}

// TimestampAtHeight must return the timestamp for the consensus state associated with the provided height.
func (l LightClientModule) TimestampAtHeight(ctx sdk.Context, clientID string, height api.Height) (uint64, error) {
	if err := validateClientID(clientID); err != nil {
		return 0, err
	}

	clientStore := l.storeProvider.ClientStore(ctx, clientID)
	clientState, found := getClientState(clientStore, l.cdc)
	if !found {
		return 0, errorsmod.Wrap(clienttypes.ErrClientNotFound, clientID)
	}

	return clientState.GetTimestampAtHeight(ctx, clientStore, l.cdc, height)
}

func validateClientID(clientID string) error {
	clientType, _, err := clienttypes.ParseClientIdentifier(clientID)
	if err != nil {
		return err
	}

	if clientType != exported.Solomachine {
		return errorsmod.Wrapf(clienttypes.ErrInvalidClientType, "expected: %s, got: %s", exported.Solomachine, clientType)
	}

	return nil
}

// // CheckSubstituteAndUpdateState must verify that the provided substitute may be used to update the subject client.
// // The light client must set the updated client and consensus states within the clientStore for the subject client.
// // DEPRECATED: will be removed as performs internal functionality
func (l LightClientModule) RecoverClient(ctx sdk.Context, clientID, substituteClientID string) error {
	return nil
}

// // Upgrade functions
// // NOTE: proof heights are not included as upgrade to a new revision is expected to pass only on the last
// // height committed by the current revision. Clients are responsible for ensuring that the planned last
// // height of the current revision is somehow encoded in the proof verification process.
// // This is to ensure that no premature upgrades occur, since upgrade plans committed to by the counterparty
// // may be cancelled or modified before the last planned height.
// // If the upgrade is verified, the upgraded client and consensus states must be set in the client store.
// // DEPRECATED: will be removed as performs internal functionality
func (l LightClientModule) VerifyUpgradeAndUpdateState(ctx sdk.Context, clientID string, newClient []byte, newConsState []byte, upgradeClientProof, upgradeConsensusStateProof []byte) error {
	return nil
}
