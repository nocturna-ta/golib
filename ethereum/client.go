package ethereum

import (
	"context"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"math/big"
)

type Client interface {
	// Connection methods
	GetEthClient() *ethclient.Client
	Close() error

	// Read methods
	GetLatestBlockNumber(ctx context.Context) (*big.Int, error)
	GetBalance(ctx context.Context, address common.Address) (*big.Int, error)
	GetTransactionByHash(ctx context.Context, hash common.Hash) (*types.Transaction, bool, error)
	GetBlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error)
	GetLogs(ctx context.Context, query ethereum.FilterQuery) ([]types.Log, error)

	// Transaction methods
	SendTransaction(ctx context.Context, tx *types.Transaction) error
	EstimateGas(ctx context.Context, call CallMsg) (uint64, error)
	SuggestGasPrice(ctx context.Context) (*big.Int, error)

	// Contract interactions
	GetCallOpts(ctx context.Context) *bind.CallOpts
	GetTransactOpts(ctx context.Context, privateKey string) (*bind.TransactOpts, error)
}

// CallMsg contains parameters for contract calls
type CallMsg struct {
	From     common.Address  // Sender address
	To       *common.Address // Recipient address (nil for contract creation)
	Gas      uint64          // Gas provided for the call
	GasPrice *big.Int        // Gas price provided for the call
	Value    *big.Int        // Amount of wei sent in the call
	Data     []byte          // Input data for the call
}
