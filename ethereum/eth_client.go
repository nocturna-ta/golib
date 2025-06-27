package ethereum

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/nocturna-ta/golib/log"
	"github.com/nocturna-ta/golib/tracing"
	"math/big"
)

type ethClient struct {
	client *ethclient.Client
}

type Options struct {
	URL string
}

func New(opts *Options) (Client, error) {
	client, err := ethclient.Dial(opts.URL)
	if err != nil {
		return nil, err
	}

	return &ethClient{
		client: client,
	}, nil
}

// GetEthClient returns the underlying ethclient.Client
func (e *ethClient) GetEthClient() bind.ContractBackend {
	return e.client
}

// Close closes the connection to the Ethereum client
func (e *ethClient) Close() error {
	if e.client != nil {
		e.client.Close()
	}
	return nil
}

// GetLatestBlockNumber returns the latest block number
func (e *ethClient) GetLatestBlockNumber(ctx context.Context) (*big.Int, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "EthClient.GetLatestBlockNumber")
	defer span.End()

	blockNumber, err := e.client.BlockNumber(ctx)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).ErrorWithCtx(ctx, "[EthClient.GetLatestBlockNumber] Failed to get latest block number")
		return nil, err
	}

	return big.NewInt(int64(blockNumber)), nil
}

// GetBalance returns the balance of the specified address
func (e *ethClient) GetBalance(ctx context.Context, address common.Address) (*big.Int, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "EthClient.GetBalance")
	defer span.End()

	balance, err := e.client.BalanceAt(ctx, address, nil)
	if err != nil {
		log.WithFields(log.Fields{
			"error":   err,
			"address": address.Hex(),
		}).ErrorWithCtx(ctx, "[EthClient.GetBalance] Failed to get balance")
		return nil, err
	}

	return balance, nil
}

// GetTransactionByHash returns the transaction for the given hash
func (e *ethClient) GetTransactionByHash(ctx context.Context, hash common.Hash) (*types.Transaction, bool, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "EthClient.GetTransactionByHash")
	defer span.End()

	tx, isPending, err := e.client.TransactionByHash(ctx, hash)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"hash":  hash.Hex(),
		}).ErrorWithCtx(ctx, "[EthClient.GetTransactionByHash] Failed to get transaction")
		return nil, false, err
	}

	return tx, isPending, nil
}

// GetBlockByNumber returns the block for the given number
func (e *ethClient) GetBlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "EthClient.GetBlockByNumber")
	defer span.End()

	block, err := e.client.BlockByNumber(ctx, number)
	if err != nil {
		log.WithFields(log.Fields{
			"error":  err,
			"number": number,
		}).ErrorWithCtx(ctx, "[EthClient.GetBlockByNumber] Failed to get block")
		return nil, err
	}

	return block, nil
}

// GetLogs returns the logs for the given filter query
func (e *ethClient) GetLogs(ctx context.Context, query ethereum.FilterQuery) ([]types.Log, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "EthClient.GetLogs")
	defer span.End()

	filterQuery := ethereum.FilterQuery{
		FromBlock: query.FromBlock,
		ToBlock:   query.ToBlock,
		Addresses: query.Addresses,
		Topics:    query.Topics,
	}

	logs, err := e.client.FilterLogs(ctx, filterQuery)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"query": query,
		}).ErrorWithCtx(ctx, "[EthClient.GetLogs] Failed to get logs")
		return nil, err
	}

	return logs, nil
}

// SendTransaction sends a transaction
func (e *ethClient) SendTransaction(ctx context.Context, tx *types.Transaction) (string, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "EthClient.SendTransaction")
	defer span.End()

	err := e.client.SendTransaction(ctx, tx)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"tx":    tx.Hash().Hex(),
		}).ErrorWithCtx(ctx, "[EthClient.SendTransaction] Failed to send transaction")
		return "", err
	}

	return tx.Hash().Hex(), nil
}

// EstimateGas estimates the gas needed to execute a call
func (e *ethClient) EstimateGas(ctx context.Context, call ethereum.CallMsg) (uint64, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "EthClient.EstimateGas")
	defer span.End()

	msg := ethereum.CallMsg{
		From:     call.From,
		To:       call.To,
		Gas:      call.Gas,
		GasPrice: call.GasPrice,
		Value:    call.Value,
		Data:     call.Data,
	}

	gas, err := e.client.EstimateGas(ctx, msg)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"call":  call,
		}).ErrorWithCtx(ctx, "[EthClient.EstimateGas] Failed to estimate gas")
		return 0, err
	}

	return gas, nil
}

// SuggestGasPrice suggests a gas price
func (e *ethClient) SuggestGasPrice(ctx context.Context) (*big.Int, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "EthClient.SuggestGasPrice")
	defer span.End()

	gasPrice, err := e.client.SuggestGasPrice(ctx)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).ErrorWithCtx(ctx, "[EthClient.SuggestGasPrice] Failed to suggest gas price")
		return nil, err
	}

	return gasPrice, nil
}

// GetCallOpts returns call options for contract calls
func (e *ethClient) GetCallOpts(ctx context.Context) *bind.CallOpts {
	return &bind.CallOpts{
		Context: ctx,
	}
}

// GetTransactOpts returns transact options for contract transactions
func (e *ethClient) GetTransactOpts(ctx context.Context, privateKey string) (*bind.TransactOpts, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "EthClient.GetTransactOpts")
	defer span.End()

	if privateKey == "" {
		return nil, errors.New("private key is required")
	}

	privateKeyECDSA, err := crypto.HexToECDSA(privateKey)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).ErrorWithCtx(ctx, "[EthClient.GetTransactOpts] Failed to convert private key")
		return nil, err
	}

	publicKey := privateKeyECDSA.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, errors.New("error casting public key to ECDSA")
	}

	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)
	nonce, err := e.client.PendingNonceAt(ctx, fromAddress)
	if err != nil {
		log.WithFields(log.Fields{
			"error":   err,
			"address": fromAddress.Hex(),
		}).ErrorWithCtx(ctx, "[EthClient.GetTransactOpts] Failed to get nonce")
		return nil, err
	}

	gasPrice, err := e.client.SuggestGasPrice(ctx)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).ErrorWithCtx(ctx, "[EthClient.GetTransactOpts] Failed to suggest gas price")
		return nil, err
	}

	chainID, err := e.client.ChainID(ctx)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).ErrorWithCtx(ctx, "[EthClient.GetTransactOpts] Failed to get chain ID")
		return nil, err
	}

	auth, err := bind.NewKeyedTransactorWithChainID(privateKeyECDSA, chainID)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).ErrorWithCtx(ctx, "[EthClient.GetTransactOpts] Failed to create transactor")
		return nil, err
	}

	auth.Nonce = big.NewInt(int64(nonce))
	auth.Value = big.NewInt(0)      // in wei
	auth.GasLimit = uint64(3000000) // in units
	auth.GasPrice = gasPrice

	return auth, nil
}
