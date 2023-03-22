package types

import (
	"encoding/json"
	"strings"
	"time"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	ibcerrors "github.com/cosmos/ibc-go/v7/internal/errors"
	"github.com/cosmos/ibc-go/v7/modules/core/exported"
)

var (
	// DefaultRelativePacketTimeoutHeight is the default packet timeout height (in blocks) relative
	// to the current block height of the counterparty chain provided by the client state. The
	// timeout is disabled when set to 0.
	DefaultRelativePacketTimeoutHeight = "0-1000"

	// DefaultRelativePacketTimeoutTimestamp is the default packet timeout timestamp (in nanoseconds)
	// relative to the current block timestamp of the counterparty chain provided by the client
	// state. The timeout is disabled when set to 0. The default is currently set to a 10 minute
	// timeout.
	DefaultRelativePacketTimeoutTimestamp = uint64((time.Duration(10) * time.Minute).Nanoseconds())
)

var _ exported.CallbackPacketData = (*FungibleTokenPacketData)(nil)

// NewFungibleTokenPacketData contructs a new FungibleTokenPacketData instance
func NewFungibleTokenPacketData(
	denom string, amount string,
	sender, receiver string,
	memo string,
) FungibleTokenPacketData {
	return FungibleTokenPacketData{
		Denom:    denom,
		Amount:   amount,
		Sender:   sender,
		Receiver: receiver,
		Memo:     memo,
	}
}

// ValidateBasic is used for validating the token transfer.
// NOTE: The addresses formats are not validated as the sender and recipient can have different
// formats defined by their corresponding chains that are not known to IBC.
func (ftpd FungibleTokenPacketData) ValidateBasic() error {
	amount, ok := sdk.NewIntFromString(ftpd.Amount)
	if !ok {
		return errorsmod.Wrapf(ErrInvalidAmount, "unable to parse transfer amount (%s) into math.Int", ftpd.Amount)
	}
	if !amount.IsPositive() {
		return errorsmod.Wrapf(ErrInvalidAmount, "amount must be strictly positive: got %d", amount)
	}
	if strings.TrimSpace(ftpd.Sender) == "" {
		return errorsmod.Wrap(ibcerrors.ErrInvalidAddress, "sender address cannot be blank")
	}
	if strings.TrimSpace(ftpd.Receiver) == "" {
		return errorsmod.Wrap(ibcerrors.ErrInvalidAddress, "receiver address cannot be blank")
	}
	return ValidatePrefixedDenom(ftpd.Denom)
}

// GetBytes is a helper for serialising
func (ftpd FungibleTokenPacketData) GetBytes() []byte {
	return sdk.MustSortJSON(mustProtoMarshalJSON(&ftpd))
}

/*

ADR-8 CallbackPacketData implementation

FungibleTokenPacketData implements CallbackPacketDataI interface. This will allow middlewares targeting specific VMs
to retrieve the desired callback addresses for the ICS20 packet on the source and destination chains.

The Memo is used to ensure that the callback is desired by the user. This allows a user to send an ICS20 packet
to a contract with ADR-8 enabled without automatically triggering the callback logic which may lead to unexpected
behaviour.

The Memo format is defined like so:

```json
{
	// ... other memo fields we don't care about
	"callbacks": {
		"src_callback_address": {contractAddrOnSrcChain},
		"dest_callback_address": {contractAddrOnDestChain},

		// optional fields
		"src_callback_msg": {jsonObjectForSrcChainCallback},
		"dest_callback_msg": {jsonObjectForDestChainCallback},
	}
}
```

For transfer, we will enforce that the src_callback_address is the same as sender and dest_callback_address is the same as receiver.
However, we may remove this restriction at a later date if it proves useful.

*/

// GetSrcCallbackAddress returns the sender address if it is also specified in
// the packet data memo. The desired callback address must be confirmed in the
// memo under the "callbacks" key. This ensures that the callback is explicitly
// desired by the user and not called automatically. If no callback address is
// specified, an empty string is returned.
//
// The memo is expected to contain the source callback address in the following format:
// { "callbacks": { "src_callback_address": {contractAddrOnSrcChain}}
//
// ADR-8 middleware should callback on the returned address if it is a PacketActor
// (i.e. smart contract that accepts IBC callbacks).
func (ftpd FungibleTokenPacketData) GetSrcCallbackAddress() string {
	if len(ftpd.Memo) == 0 {
		return ""
	}

	jsonObject := make(map[string]interface{})
	err := json.Unmarshal([]byte(ftpd.Memo), &jsonObject)
	if err != nil {
		return ""
	}

	callbackData, ok := jsonObject["callbacks"].(map[string]interface{})
	if !ok {
		return ""
	}

	if callbackData["src_callback_address"] == ftpd.Sender {
		return ftpd.Sender
	}

	return ""
}

// GetDestCallbackAddress returns the receiving address if it is also specified in
// the packet data memo. The desired callback address must be confirmed in the
// memo under the "callbacks" key. This ensures that the callback is explicitly
// desired by the user and not called automatically. If no callback address is
// specified, an empty string is returned.
//
// The memo is expected to contain the destination callback address in the following format:
// { "callbacks": { "dest_callback_address": {contractAddrOnDestChain}}
//
// ADR-8 middleware should callback on the returned address if it is a PacketActor
// (i.e. smart contract that accepts IBC callbacks).
func (ftpd FungibleTokenPacketData) GetDestCallbackAddress() string {
	if len(ftpd.Memo) == 0 {
		return ""
	}

	jsonObject := make(map[string]interface{})
	err := json.Unmarshal([]byte(ftpd.Memo), &jsonObject)
	if err != nil {
		return ""
	}

	callbackData, ok := jsonObject["callbacks"].(map[string]interface{})
	if !ok {
		return ""
	}

	if callbackData["dest_callback_address"] == ftpd.Receiver {
		return ftpd.Receiver
	}

	return ""
}

// UserDefinedGasLimit returns 0 (no-op). The gas limit of the executing
// transaction will be used.
func (fptd FungibleTokenPacketData) UserDefinedGasLimit() uint64 {
	return 0
}
