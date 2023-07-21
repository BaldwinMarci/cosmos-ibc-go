package types

import (
	"fmt"

	errorsmod "cosmossdk.io/errors"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v7/modules/core/exported"
)

// CheckSubstituteAndUpdateState will try to update the client with the state of the
// substitute.
func (cs ClientState) CheckSubstituteAndUpdateState(
	ctx sdk.Context,
	_ codec.BinaryCodec,
	subjectClientStore, substituteClientStore sdk.KVStore,
	substituteClient exported.ClientState,
) error {
	var (
		SubjectPrefix    = []byte("subject/")
		SubstitutePrefix = []byte("substitute/")
	)

	_, ok := substituteClient.(*ClientState)
	if !ok {
		return errorsmod.Wrapf(
			clienttypes.ErrInvalidClient,
			fmt.Sprintf("invalid substitute client state. expected type %T, got %T", &ClientState{}, substituteClient),
		)
	}

	store := newUpdateProposalWrappedStore(subjectClientStore, substituteClientStore, SubjectPrefix, SubstitutePrefix)

	payload := SudoMsg{
		CheckSubstituteAndUpdateState: &checkSubstituteAndUpdateStateMsg{},
	}

	_, err := call[contractResult](ctx, store, &cs, payload)
	return err
}
