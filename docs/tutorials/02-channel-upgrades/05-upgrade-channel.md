---
title: Upgrade channel
sidebar_label: Upgrade channel
sidebar_position: 5
slug: /channel-upgrades/upgrade-channel
---

# Upgrade the ICS 20 transfer channel

## Start the relayer

We start the relayer:

```bash
hermes --config config.toml start
```

## Initiate the upgrade

The [initiation of the upgrade process is authority-gated](https://ibc.cosmos.network/main/ibc/channel-upgrades#governance-gating-on-chanupgradeinit), so in this example the message to initiate the upgrade will execute when a governance proposal passes. The contents of the governance proposal are:

```json title=proposal.json
{
  "title": "Channel upgrade init",
  "summary": "Channel upgrade init",
  "messages": [
    {
      "@type": "/ibc.core.channel.v1.MsgChannelUpgradeInit",
      "signer": "cosmos10d07y265gmmuvt4z0w9aw880jnsr700j6zn9kn",
      "port_id": "transfer",
      "channel_id": "channel-0",
      "fields": {
        "ordering": "ORDER_UNORDERED",
        "connection_hops": ["connection-0"],
        "version": "{\"fee_version\":\"ics29-1\",\"app_version\":\"ics20-1\"}"
      }
    }
  ],
  "metadata": "AQ==",
  "deposit": "100005stake"
}
```

where `cosmos10d07y265gmmuvt4z0w9aw880jnsr700j6zn9kn` is the address of the governance module on `chain1`. The upgrade will modify the channel version to include the fee version.

We submit the proposal:

```bash
simd tx gov submit-proposal ./proposal_upgrade_channel.json --from cosmos1vdy5fp0jy2l2ees870a7mls357v7uad6ufzcyz \
--chain-id chain1 \
--keyring-backend test \
--home ../../gm/chain1 \
--node http://localhost:27000
```

Now we vote for the proposal:

```bash
simd tx gov vote 1 yes \
--from cosmos18phmkrpnn6gmpzscf6hnf5zpv06sygxc6f2v92 \
--chain-id chain1 \
--keyring-backend test \
--home ../../gm/chain1 \
--node http://localhost:27000
```

And we wait for the voting period to end. Once it ends we can check that the proposal has passed (i.e. the status has changed from `PROPOSAL_STATUS_VOTING_PERIOD` to `PROPOSAL_STATUS_PASSED`):

```bash
simd q gov proposals --node http://localhost:27000
```

```yaml
pagination:
  total: "1"
proposals:
- deposit_end_time: "2024-01-27T21:29:52.430508Z"
  final_tally_result:
    abstain_count: "0"
    no_count: "0"
    no_with_veto_count: "0"
    yes_count: "1000000"
  id: "1"
  messages:
  - type: /ibc.core.channel.v1.MsgChannelUpgradeInit
    value:
      channel_id: channel-0
      fields: {}
      port_id: transfer
      signer: cosmos10d07y265gmmuvt4z0w9aw880jnsr700j6zn9kn
  metadata: AQ==
  proposer: cosmos1vdy5fp0jy2l2ees870a7mls357v7uad6ufzcyz
  status: 3
  submit_time: "2024-01-25T21:29:52.430508Z"
  summary: Channel upgrade init
  title: Channel upgrade init
  total_deposit:
  - amount: "100005"
    denom: stake
  voting_end_time: "2024-01-25T21:32:52.430508Z"
  voting_start_time: "2024-01-25T21:29:52.430508Z"
```

Now we wait for the relayer to complete the upgrade handshake.

## Check ugprade completed

Once the handshake has completed we verify that the channel has successfully upgraded:

```bash
simd q ibc channel channels --node http://localhost:27000
```

```yaml
channels:
- channel_id: channel-0
  connection_hops:
  - connection-0
  counterparty:
    channel_id: channel-0
    port_id: transfer
  ordering: ORDER_UNORDERED
  port_id: transfer
  state: STATE_OPEN
  upgrade_sequence: "1"
  version: '{"fee_version":"ics29-1","app_version":"ics20-1"}'
height:
  revision_height: "135"
  revision_number: "0"
pagination:
  next_key: null
  total: "0"
```

The channel version on `chain1` is what we expect.

```bash
simd q ibc-fee channels --node http://localhost:27000
```

```yaml
fee_enabled_channels:
- channel_id: channel-0
  port_id: transfer
pagination:
  next_key: null
  total: "0"
```

As we expect there is one incentivized channel.

```bash
simd q ibc channel channels --node http://localhost:27010
```

```yaml
channels:
- channel_id: channel-0
  connection_hops:
  - connection-0
  counterparty:
    channel_id: channel-0
    port_id: transfer
  ordering: ORDER_UNORDERED
  port_id: transfer
  state: STATE_OPEN
  upgrade_sequence: "1"
  version: '{"fee_version":"ics29-1","app_version":"ics20-1"}'
height:
  revision_height: "138"
  revision_number: "0"
pagination:
  next_key: null
  total: "0"
```

The channel version on `chain2` is also what we expect.

```bash
simd q ibc-fee channels --node http://localhost:27010
```

```yaml
fee_enabled_channels:
- channel_id: channel-0
  port_id: transfer
pagination:
  next_key: null
  total: "0"
```

As we expect there is one incentivized channel as well on `chain2`.

From now ICS 20 packets sent on this channel can be incentivized.
