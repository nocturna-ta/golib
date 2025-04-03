package ethereum

import (
	"context"
	"fmt"
	"github.com/avast/retry-go/v4"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/nocturna-ta/golib/custerr"
	"github.com/nocturna-ta/golib/log"
	"github.com/nocturna-ta/golib/response"
	"github.com/nocturna-ta/golib/tracing"
	"math/big"
	"time"
)

type GasOptions struct {
	GasLimit  uint64
	GasPrice  *big.Int
	GasTipCap *big.Int
	GasFeeCap *big.Int
}

type RetryConfig struct {
	MaxRetries int
	RetryDelay time.Duration
	MaxJitter  time.Duration
}

type ClientConfig struct {
	URL               string
	ChainID           int64
	RetryConfig       *RetryConfig
	DefaultGasOptions *GasOptions
}

type Module struct {
	ethClient     *ethclient.Client
	rpcClient     *rpc.Client
	config        *ClientConfig
	retryConfig   *RetryConfig
	chainID       *big.Int
	defaultGasOps *GasOptions
}

type Opts struct {
	EthClient         *ethclient.Client
	RPCClient         *rpc.Client
	Config            *ClientConfig
	RetryConfig       *RetryConfig
	ChainID           *big.Int
	DefaultGasOptions *GasOptions
}

type Clients interface {
	Close()
	ChainID() *big.Int
	BlockNumber(ctx context.Context) (uint64, error)
	GetBalance(ctx context.Context, address common.Address, blockNumber *big.Int) (*big.Int, error)
	GetTransaction(ctx context.Context, txHash common.Hash) (*types.Transaction, bool, error)
	GetTransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error)
	WaitForReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error)
	SendTransaction(ctx context.Context, tx *types.Transaction) error
	EstimateGas(ctx context.Context, callMsg ethereum.CallMsg) (uint64, error)
	GetEthClient() *ethclient.Client
	GetRpcClient() *rpc.Client
}

func NewClient(opts *Opts) Clients {
	return &Module{
		ethClient:     opts.EthClient,
		rpcClient:     opts.RPCClient,
		config:        opts.Config,
		retryConfig:   opts.RetryConfig,
		chainID:       opts.ChainID,
		defaultGasOps: opts.DefaultGasOptions,
	}

}

func (c *Module) Close() {
	c.ethClient.Close()
}

// ChainID returns the chain ID of the connected Ethereum network
func (c *Module) ChainID() *big.Int {
	return c.chainID
}

// BlockNumber returns the latest block number
func (c *Module) BlockNumber(ctx context.Context) (uint64, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "EthClient.BlockNumber")
	defer span.End()

	var blockNumber uint64
	err := retry.Do(
		func() error {
			var err error
			blockNumber, err = c.ethClient.BlockNumber(ctx)
			return err
		},
		retry.Attempts(uint(c.retryConfig.MaxRetries)),
		retry.Delay(c.retryConfig.RetryDelay),
		retry.MaxJitter(c.retryConfig.MaxJitter),
		retry.OnRetry(func(n uint, err error) {
			log.WithFields(log.Fields{
				"attempt": n + 1,
				"error":   err,
			}).WarnWithCtx(ctx, "Retrying Ethereum BlockNumber request")
		}),
	)

	if err != nil {
		return 0, &custerr.ErrChain{
			Message: "failed to get latest block number",
			Cause:   err,
			Code:    500,
			Type:    response.ErrInternalServerError,
		}
	}

	return blockNumber, nil
}

// GetBalance returns the balance of an account
func (c *Module) GetBalance(ctx context.Context, address common.Address, blockNumber *big.Int) (*big.Int, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "EthClient.GetBalance")
	defer span.End()

	var balance *big.Int
	err := retry.Do(
		func() error {
			var err error
			balance, err = c.ethClient.BalanceAt(ctx, address, blockNumber)
			return err
		},
		retry.Attempts(uint(c.retryConfig.MaxRetries)),
		retry.Delay(c.retryConfig.RetryDelay),
		retry.MaxJitter(c.retryConfig.MaxJitter),
		retry.OnRetry(func(n uint, err error) {
			log.WithFields(log.Fields{
				"attempt": n + 1,
				"error":   err,
				"address": address.Hex(),
			}).WarnWithCtx(ctx, "Retrying Ethereum GetBalance request")
		}),
	)

	if err != nil {
		return nil, &custerr.ErrChain{
			Message: fmt.Sprintf("failed to get balance for address %s", address.Hex()),
			Cause:   err,
			Code:    500,
			Type:    response.ErrInternalServerError,
		}
	}

	return balance, nil
}

// GetTransaction returns a transaction by its hash
func (c *Module) GetTransaction(ctx context.Context, txHash common.Hash) (*types.Transaction, bool, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "EthClient.GetTransaction")
	defer span.End()

	var tx *types.Transaction
	var isPending bool
	err := retry.Do(
		func() error {
			var err error
			tx, isPending, err = c.ethClient.TransactionByHash(ctx, txHash)
			return err
		},
		retry.Attempts(uint(c.retryConfig.MaxRetries)),
		retry.Delay(c.retryConfig.RetryDelay),
		retry.MaxJitter(c.retryConfig.MaxJitter),
		retry.OnRetry(func(n uint, err error) {
			log.WithFields(log.Fields{
				"attempt": n + 1,
				"error":   err,
				"txHash":  txHash.Hex(),
			}).WarnWithCtx(ctx, "Retrying Ethereum GetTransaction request")
		}),
	)

	if err != nil {
		return nil, false, &custerr.ErrChain{
			Message: fmt.Sprintf("failed to get transaction %s", txHash.Hex()),
			Cause:   err,
			Code:    500,
			Type:    response.ErrInternalServerError,
		}
	}

	return tx, isPending, nil
}

// GetTransactionReceipt returns a transaction receipt by its hash
func (c *Module) GetTransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "EthClient.GetTransactionReceipt")
	defer span.End()

	var receipt *types.Receipt
	err := retry.Do(
		func() error {
			var err error
			receipt, err = c.ethClient.TransactionReceipt(ctx, txHash)
			return err
		},
		retry.Attempts(uint(c.retryConfig.MaxRetries)),
		retry.Delay(c.retryConfig.RetryDelay),
		retry.MaxJitter(c.retryConfig.MaxJitter),
		retry.OnRetry(func(n uint, err error) {
			log.WithFields(log.Fields{
				"attempt": n + 1,
				"error":   err,
				"txHash":  txHash.Hex(),
			}).WarnWithCtx(ctx, "Retrying Ethereum GetTransactionReceipt request")
		}),
	)

	if err != nil {
		return nil, &custerr.ErrChain{
			Message: fmt.Sprintf("failed to get transaction receipt for %s", txHash.Hex()),
			Cause:   err,
			Code:    500,
			Type:    response.ErrInternalServerError,
		}
	}

	return receipt, nil
}

// WaitForReceipt waits for a transaction to be mined and returns its receipt
func (c *Module) WaitForReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "EthClient.WaitForReceipt")
	defer span.End()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		receipt, err := c.GetTransactionReceipt(ctx, txHash)
		if err == nil {
			return receipt, nil
		}

		// Check if the error is "not found", which is expected while waiting
		if err, ok := err.(*custerr.ErrChain); ok {
			if err.Cause == ethereum.NotFound {
				// This is normal while waiting for the transaction to be mined
				log.WithFields(log.Fields{
					"txHash": txHash.Hex(),
				}).DebugWithCtx(ctx, "Transaction not yet mined, waiting...")
			} else {
				// Return unexpected errors
				return nil, err
			}
		}

		select {
		case <-ctx.Done():
			return nil, &custerr.ErrChain{
				Message: "context cancelled while waiting for transaction receipt",
				Cause:   ctx.Err(),
				Code:    499, // Client closed request
				Type:    response.ErrTimeoutError,
			}
		case <-ticker.C:
			// Continue the loop
		}
	}
}

// SendTransaction sends a signed transaction to the network
func (c *Module) SendTransaction(ctx context.Context, tx *types.Transaction) error {
	span, ctx := tracing.StartSpanFromContext(ctx, "EthClient.SendTransaction")
	defer span.End()

	err := retry.Do(
		func() error {
			return c.ethClient.SendTransaction(ctx, tx)
		},
		retry.Attempts(uint(c.retryConfig.MaxRetries)),
		retry.Delay(c.retryConfig.RetryDelay),
		retry.MaxJitter(c.retryConfig.MaxJitter),
		retry.OnRetry(func(n uint, err error) {
			log.WithFields(log.Fields{
				"attempt": n + 1,
				"error":   err,
				"txHash":  tx.Hash().Hex(),
			}).WarnWithCtx(ctx, "Retrying Ethereum SendTransaction request")
		}),
	)

	if err != nil {
		return &custerr.ErrChain{
			Message: "failed to send transaction",
			Cause:   err,
			Code:    500,
			Type:    response.ErrInternalServerError,
		}
	}

	return nil
}

// EstimateGas estimates the gas needed to execute a transaction
func (c *Module) EstimateGas(ctx context.Context, call ethereum.CallMsg) (uint64, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "EthClient.EstimateGas")
	defer span.End()

	var gas uint64
	err := retry.Do(
		func() error {
			var err error
			gas, err = c.ethClient.EstimateGas(ctx, call)
			return err
		},
		retry.Attempts(uint(c.retryConfig.MaxRetries)),
		retry.Delay(c.retryConfig.RetryDelay),
		retry.MaxJitter(c.retryConfig.MaxJitter),
		retry.OnRetry(func(n uint, err error) {
			log.WithFields(log.Fields{
				"attempt": n + 1,
				"error":   err,
				"from":    call.From.Hex(),
				"to":      call.To.Hex(),
			}).WarnWithCtx(ctx, "Retrying Ethereum EstimateGas request")
		}),
	)

	if err != nil {
		return 0, &custerr.ErrChain{
			Message: "failed to estimate gas",
			Cause:   err,
			Code:    500,
			Type:    response.ErrInternalServerError,
		}
	}

	return gas, nil
}

// GetEthClient returns the underlying ethclient.Client
func (c *Module) GetEthClient() *ethclient.Client {
	return c.ethClient
}
func (c *Module) GetRpcClient() *rpc.Client {
	//TODO implement me
	panic("implement me")
}
