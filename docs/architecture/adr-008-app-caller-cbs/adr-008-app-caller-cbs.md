# ADR 008: Callback to IBC Actors

## Changelog
* 2022-08-10: Initial Draft

## Status

Proposed

## Context

IBC was designed with callbacks between core IBC and IBC applications. IBC apps would send a packet to core IBC. When the result of the packet lifecycle eventually resolved into either an acknowledgement or a timeout, core IBC called a callback on the IBC application so that the IBC application could take action on the basis of the result (e.g. unescrow tokens for ICS-20).

This setup worked well for off-chain users interacting with IBC applications.

We are now seeing the desire for secondary applications (e.g. smart contracts, modules) to call into IBC apps as part of their state machine logic and then do some actions on the basis of the packet result. Or to receive a packet from IBC and do some logic upon receipt.

Example Usecases:
- Send an ICS-20 packet, and if it is successful, then send an ICA-packet to swap tokens on LP and return funds to sender
- Execute some logic upon receipt of token transfer to a smart contract address

This requires a second layer of callbacks. The IBC application already gets the result of the packet from core IBC, but currently there is no standardized way to pass this information on to an actor module/smart contract.

## Definitions

- Actor: an actor is an on-chain module (this may be a hardcoded module in the chain binary or a smart contract) that wishes to execute custom logic whenever IBC receives a packet flow that it has either sent or received. It **must** be addressable by a string value.

## Decision

Create a standardized callback interface that actors can implement. IBC applications (or middleware that wraps IBC applications) can now call this callback to route the result of the packet/channel handshake from core IBC to the IBC application to the original actor on the sending chain. IBC applications can route the packet receipt to the destination actor on the receiving chain.

IBC actors may implement the following interface:

```go
type IBCActor interface {
    // OnChannelOpen will be called on the IBCActor when the channel opens
    // this will happen either on ChanOpenAck or ChanOpenConfirm
    OnChannelOpen(ctx sdk.Context, portID, channelID, version string)

    // OnChannelClose will be called on the IBCActor if the channel closes
    // this will be called on either ChanCloseInit or ChanCloseConfirm and if the channel handshake fails on our end
    // NOTE: currently the channel does not automatically close if the counterparty fails the handhshake so actors must be prepared for an OpenInit to never return a callback for the time being
    OnChannelClose(ctx sdk.Context, portID, channelID string)

    // IBCActor must also implement PacketActor interface
    PacketActor
}

// PacketActor is split out into its own separate interface since implementors may choose
// to only support callbacks for packet methods rather than supporting the full IBCActor interface
type PacketActor interface {
    // OnRecvPacket will be called on the IBCActor after the IBC Application
    // handles the RecvPacket callback if the packet has an IBC Actor as a receiver.
    OnRecvPacket(ctx sdk.Context, packet channeltypes.Packet, relayer string) error

    // OnAcknowledgementPacket will be called on the IBC Actor
    // after the IBC Application handles its own OnAcknowledgementPacket callback
    OnAcknowledgmentPacket(
        ctx sdk.Context,
        packet channeltypes.Packet,
        ack exported.Acknowledgement,
        relayer string
    ) error

    // OnTimeoutPacket will be called on the IBC Actor
    // after the IBC Application handles its own OnTimeoutPacket callback
    OnTimeoutPacket(
        ctx sdk.Context,
        packet channeltypes.Packet,
        relayer string
    ) error
}
```

The CallbackPacketData interface will get extended to add `GetSrcCallbackAddress` and `GetDestCallbackAddress` methods. These may return an address
or they may return the empty string. The address may reference an IBCActor or it may be a regular user address. If the address is not an IBCActor, the actor callback must continue processing (no-op). Any IBC application or middleware that uses these methods must handle these cases. In most cases, the `GetSrcCallbackAddress` will be the sender address and the `GetDestCallbackAddress` will be the receiver address. However, these are named generically so that implementors may choose a different contract address for the callback if they choose.

```go
// Implemented by any packet data type that wants to support
// PacketActor callbacks
type CallbackPacketData interface {
    // may return the empty string
    GetSrcCallbackAddress() string

    // may return the empty string
    GetDestCallbackAddress() string
}
```

IBC Apps or middleware can then call the IBCActor callbacks like so in their own callbacks:

### Handshake Callbacks

The handshake init callbacks (`OnChanOpenInit` and `OnChanCloseInit`) will need to include an additional field so that the initiating actor can be tracked and called upon during handshake completion.

```go
func OnChanOpenInit(
    ctx sdk.Context,
    order channeltypes.Order,
    connectionHops []string,
    portID string,
    channelID string,
    channelCap *capabilitytypes.Capability,
    counterparty channeltypes.Counterparty,
    version string,
    actor string,
) (string, error) {
    acc := k.getAccount(ctx, actor)
    ibcActor, ok := acc.(IBCActor)
    if ok {
        k.setActor(ctx, portID, channelID, actor)
    }
    
    // continued logic
}

func OnChanOpenAck(
    ctx sdk.Context,
    portID,
    channelID string,
    counterpartyChannelID string,
    counterpartyVersion string,
) error {
    // run any necessary logic first
    // negotiate final version

    actor := k.getActor(ctx, portID, channelID)
    if actor != "" {
        ibcActor, _ := acc.(IBCActor)
        ibcActor.OnChanOpen(ctx, portID, channelID, version)
    }
    // cleanup state
    k.deleteActor(ctx, portID, channelID)
}

func OnChanOpenConfirm(
    ctx sdk.Context,
    portID,
    channelID string,
) error {
    // run any necesssary logic first
    // retrieve final version

    actor := k.getActor(ctx, portID, channelID)
    if actor != "" {
        ibcActor, _ := acc.(IBCActor)
        ibcActor.OnChanOpen(ctx, portID, channelID, version)
    }
    // cleanup state
    k.deleteActor(ctx, portID, channelID)
}

func OnChanCloseInit(
    ctx sdk.Context,
    portID,
    channelID,
    actor string,
) error {
    acc := k.getAccount(ctx, actor)
    ibcActor, ok := acc.(IBCActor)
    if ok {
        k.setActor(ctx, portID, channelID, actor)
    }
    
    // continued logic
}

func OnChanCloseConfirm(
    ctx sdk.Context,
    portID,
    channelID string,
) error {
    // run any necesssary logic first

    actor := k.getActor(ctx, portID, channelID)
    if actor != "" {
        ibcActor, _ := acc.(IBCActor)
        ibcActor.OnChanClose(ctx, portID, channelID)
    }
    // cleanup state
    k.deleteActor(ctx, portID, channelID)
}
```

### PacketCallbacks

No packet callback API will need to change.

```go
// Call the IBCActor recvPacket callback after processing the packet
// if the recvPacket callback exists and returns an error
// then return an error ack to revert all packet data processing
func OnRecvPacket(
    ctx sdk.Context,
    packet channeltypes.Packet,
    relayer sdk.AccAddress,
) exported.Acknowledgement {
    // run any necesssary logic first
    // IBCActor logic will postprocess

    acc := k.getAccount(ctx, packet.GetDstCallbackAddress())
    ibcActor, ok := acc.(IBCActor)
    if ok {
        err := ibcActor.OnRecvPacket(ctx, packet, relayer)
        if err != nil {
            return AcknowledgementError(err)
        }
    }
}

// Call the IBCActor acknowledgementPacket callback after processing the packet
// if the ackPacket callback exists and returns an error
// DO NOT return the error upstream. The acknowledgement must complete for the packet
// lifecycle to end, so the custom callback cannot block completion.
// Instead we emit error events and set the error in state
// so that users and on-chain logic can handle this appropriately
func (im IBCModule) OnAcknowledgementPacket(
    ctx sdk.Context,
    packet channeltypes.Packet,
    acknowledgement []byte,
    relayer string,
) error {
    // application-specific onAcknowledgmentPacket logic

    // unmarshal ack bytes into the acknowledgment interface
    var ack exported.Acknowledgement
    unmarshal(acknowledgement, ack)

    // send acknowledgement to original actor
    acc := k.getAccount(ctx, packet.GetSrcCallbackAddress())
    ibcActor, ok := acc.(IBCActor)
    if ok {
        err := ibcActor.OnAcknowledgementPacket(ctx, packet, ack, relayer)
        setAckCallbackError(ctx, packet, err)
        emitAckCallbackErrorEvents(err)
    }
}

// Call the IBCActor timeoutPacket callback after processing the packet
// if the timeoutPacket callback exists and returns an error
// DO NOT return the error upstream. The timeout must complete for the packet
// lifecycle to end, so the custom callback cannot block completion.
// Instead we emit error events and set the error in state
// so that users and on-chain logic can handle this appropriately
func (im IBCModule) OnTimeoutPacket(
    ctx sdk.Context,
    packet channeltypes.Packet,
    relayer string,
) error {
    // application-specific onTimeoutPacket logic

    // call timeout callback on original actor
    acc := k.getAccount(ctx, packet.GetSrcCallbackAddress())
    ibcActor, ok := acc.(IBCActor)
    if ok {
        err := ibcActor.OnTimeoutPacket(ctx, packet, relayer)
        setTimeoutCallbackError(ctx, packet, err)
        emitTimeoutCallbackErrorEvents(err)
    }
}
```

## Consequences

### Positive

- IBC Actors can now programatically execute logic that involves sending a packet and then performing some additional logic once the packet lifecycle is complete
- Leverages the same callback architecture used between core IBC and IBC applications

### Negative

- Callbacks may now have unbounded gas consumption since the actor may execute arbitrary logic. Chains implementing this feature should take care to place limitations on how much gas an actor callback can consume.

### Neutral

## References

- https://github.com/cosmos/ibc-go/issues/1660
