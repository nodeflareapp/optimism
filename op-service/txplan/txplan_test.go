package txplan

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

type recordingEstimator struct {
	msg ethereum.CallMsg
}

func (e *recordingEstimator) EstimateGas(_ context.Context, msg ethereum.CallMsg) (uint64, error) {
	e.msg = msg
	return 100_000, nil
}

func TestPlannedTx_Defaults(t *testing.T) {
	key, err := crypto.GenerateKey()
	require.NoError(t, err)

	ptx := NewPlannedTx(WithPrivateKey(key), WithValue(eth.WeiU64(123)))
	t.Log("tx", ptx.Signed.String())

	block := types.NewBlock(&types.Header{BaseFee: big.NewInt(7e9)}, nil, nil, nil, types.DefaultBlockConfig)
	blockInfo := eth.BlockToInfo(block)
	ptx.AgainstBlock.Set(blockInfo)

	expectedAddr := crypto.PubkeyToAddress(key.PublicKey)
	signer := types.LatestSignerForChainID(big.NewInt(1))
	{
		tx, err := ptx.Signed.Eval(context.Background())
		require.NoError(t, err)

		sender, err := signer.Sender(tx)
		require.NoError(t, err)
		require.Equal(t, expectedAddr, sender)

		require.Equal(t, big.NewInt(123), tx.Value())
	}

	// Get a new signed tx
	ptx.Value.Set(big.NewInt(42))
	{
		tx, err := ptx.Signed.Eval(context.Background())
		require.NoError(t, err)

		sender, err := signer.Sender(tx)
		require.NoError(t, err)
		require.Equal(t, expectedAddr, sender)

		require.Equal(t, big.NewInt(42), tx.Value())
	}
}

func TestWithEstimator_UsesFeeCapAsGasPrice(t *testing.T) {
	estimator := new(recordingEstimator)
	feeCap := big.NewInt(9e9)
	tipCap := big.NewInt(2e9)
	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	ptx := NewPlannedTx(
		WithEstimator(estimator, false),
		WithGasFeeCap(feeCap),
		WithGasTipCap(tipCap),
		WithPrivateKey(key),
	)
	block := types.NewBlock(&types.Header{BaseFee: big.NewInt(7e9)}, nil, nil, nil, types.DefaultBlockConfig)
	ptx.AgainstBlock.Set(eth.BlockToInfo(block))

	gas, err := ptx.Gas.Eval(context.Background())
	require.NoError(t, err)
	require.Equal(t, uint64(100_000), gas)
	require.Equal(t, feeCap, estimator.msg.GasPrice)
	require.Nil(t, estimator.msg.GasFeeCap)
	require.Nil(t, estimator.msg.GasTipCap)
}
