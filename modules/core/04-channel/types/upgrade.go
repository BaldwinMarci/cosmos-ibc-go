package types

import (
	"errors"
	"fmt"
	"strings"

	errorsmod "cosmossdk.io/errors"

	"github.com/cosmos/ibc-go/v7/internal/collections"
)

// NewUpgrade creates a new Upgrade instance.
func NewUpgrade(upgradeFields UpgradeFields, timeout Timeout, latestPacketSent uint64) Upgrade {
	return Upgrade{
		Fields:             upgradeFields,
		Timeout:            timeout,
		LatestSequenceSend: latestPacketSent,
	}
}

// NewUpgradeFields returns a new ModifiableUpgradeFields instance.
func NewUpgradeFields(ordering Order, connectionHops []string, version string) UpgradeFields {
	return UpgradeFields{
		Ordering:       ordering,
		ConnectionHops: connectionHops,
		Version:        version,
	}
}

// ValidateBasic performs a basic validation of the upgrade fields
func (u Upgrade) ValidateBasic() error {
	if err := u.Fields.ValidateBasic(); err != nil {
		return errorsmod.Wrap(err, "proposed upgrade fields are invalid")
	}

	if !u.Timeout.IsValid() {
		return errorsmod.Wrap(ErrInvalidUpgrade, "upgrade timeout height and upgrade timeout timestamp cannot both be 0")
	}

	return nil
}

// ValidateBasic performs a basic validation of the proposed upgrade fields
func (uf UpgradeFields) ValidateBasic() error {
	if !collections.Contains(uf.Ordering, []Order{ORDERED, UNORDERED}) {
		return errorsmod.Wrap(ErrInvalidChannelOrdering, uf.Ordering.String())
	}

	if len(uf.ConnectionHops) != 1 {
		return errorsmod.Wrap(ErrTooManyConnectionHops, "current IBC version only supports one connection hop")
	}

	if strings.TrimSpace(uf.Version) == "" {
		return errorsmod.Wrap(ErrInvalidChannelVersion, "version cannot be empty")
	}

	return nil
}

// UpgradeError defines an error that occurs during an upgrade.
type UpgradeError struct {
	// err is the underlying error that caused the upgrade to fail.
	// this error should not be written to state.
	err error
	// sequence is the upgrade sequence number of the upgrade that failed.
	sequence uint64
}

var _ error = &UpgradeError{}

// NewUpgradeError returns a new UpgradeError instance.
func NewUpgradeError(upgradeSequence uint64, err error) *UpgradeError {
	return &UpgradeError{
		err:      err,
		sequence: upgradeSequence,
	}
}

// Error implements the error interface, returning the underlying error which caused the upgrade to fail.
func (u *UpgradeError) Error() string {
	return u.err.Error()
}

// Is returns true if the underlying error is of the given err type.
func (u *UpgradeError) Is(err error) bool {
	return errors.Is(u.err, err)
}

// Unwrap returns the base error that caused the upgrade to fail.
func (u *UpgradeError) Unwrap() error {
	baseError := u.err
	for {
		if err := errors.Unwrap(baseError); err != nil {
			baseError = err
		} else {
			return baseError
		}
	}
}

// Cause implements the sdk error interface which uses this function to unwrap the error in various functions such as `wrappedError.Is()`.
// Cause returns the underlying error which caused the upgrade to fail.
func (u *UpgradeError) Cause() error {
	return u.err
}

// GetErrorReceipt returns an error receipt with the code from the underlying error type stripped.
func (u *UpgradeError) GetErrorReceipt() ErrorReceipt {
	// restoreErrorString defines a string constant included in error receipts.
	// NOTE: Changing this const is state machine breaking as it is written into state.
	const restoreErrorString = "restored channel to pre-upgrade state"

	_, code, _ := errorsmod.ABCIInfo(u, false) // discard non-determinstic codespace and log values
	return ErrorReceipt{
		Sequence: u.sequence,
		Message:  fmt.Sprintf("ABCI code: %d: %s", code, restoreErrorString),
	}
}
