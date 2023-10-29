package types_test

import (
	"encoding/base64"
	"encoding/json"

	wasmvm "github.com/CosmWasm/wasmvm"
	wasmvmtypes "github.com/CosmWasm/wasmvm/types"

	errorsmod "cosmossdk.io/errors"

	wasmtesting "github.com/cosmos/ibc-go/modules/light-clients/08-wasm/testing"
	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/types"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"
)

func (suite *TypesTestSuite) TestExportGenesisGrandpa() {
	suite.SetupWasmGrandpa()

	clientStateData, err := base64.StdEncoding.DecodeString(suite.testData["client_state_data"])
	suite.Require().NoError(err)

	clientState := types.NewClientState(clientStateData, suite.codeHash, clienttypes.NewHeight(2000, 4))
	gm := clientState.ExportMetadata(suite.store)
	suite.Require().NotNil(gm, "client returned nil")
	suite.Require().Len(gm, 0, "exported metadata has unexpected length")
}

func (suite *TypesTestSuite) TestExportMetatada() {
	mockMetadata := clienttypes.NewGenesisMetadata([]byte("key"), []byte("value"))

	testCases := []struct {
		name        string
		malleate    func()
		expPanic    error
		expMetadata []exported.GenesisMetadata
	}{
		{
			"success",
			func() {
				suite.mockVM.RegisterQueryCallback(types.ExportMetadataMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, queryMsg []byte, _ wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) ([]byte, uint64, error) {
					var msg types.QueryMsg

					err := json.Unmarshal(queryMsg, &msg)
					suite.Require().NoError(err)

					suite.Require().NotNil(msg.ExportMetadata)
					suite.Require().Nil(msg.VerifyClientMessage)
					suite.Require().Nil(msg.Status)
					suite.Require().Nil(msg.CheckForMisbehaviour)
					suite.Require().Nil(msg.TimestampAtHeight)

					resp, err := json.Marshal(types.ExportMetadataResult{
						GenesisMetadata: []clienttypes.GenesisMetadata{mockMetadata},
					})
					suite.Require().NoError(err)

					return resp, wasmtesting.DefaultGasUsed, nil
				})
			},
			nil,
			[]exported.GenesisMetadata{mockMetadata},
		},
		{
			"failure: contract returns an error",
			func() {
				suite.mockVM.RegisterQueryCallback(types.ExportMetadataMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, queryMsg []byte, _ wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) ([]byte, uint64, error) {
					return nil, 0, wasmtesting.ErrMockContract
				})
			},
			errorsmod.Wrapf(types.ErrWasmContractCallFailed, "query to wasm contract failed: %v", wasmtesting.ErrMockContract),
			nil,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupWasmWithMockVM()

			endpoint := wasmtesting.NewWasmEndpoint(suite.chainA)
			err := endpoint.CreateClient()
			suite.Require().NoError(err)

			clientState := endpoint.GetClientState()

			tc.malleate()

			store := suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.chainA.GetContext(), endpoint.ClientID)

			var metadata []exported.GenesisMetadata
			exportMetadata := func() {
				metadata = clientState.ExportMetadata(store)
			}

			if tc.expPanic == nil {
				exportMetadata()

				suite.Require().Equal(tc.expMetadata, metadata)
			} else {
				suite.Require().PanicsWithError(tc.expPanic.Error(), exportMetadata)
			}
		})
	}
}
