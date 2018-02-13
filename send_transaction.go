// Copyright 2016-2017 Hyperchain Corp.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"errors"
	"math/big"
	"os"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/rjl493456442/ethclient/client"
	"gopkg.in/urfave/cli.v1"
)

var (
	errInvalidArguments  = errors.New("invalid transaction or call arguments")
	errWaitTimeout       = errors.New("wait transaction mined timeout")
	errInvalidBatchIndex = errors.New("invalid batch index")
)

var commandSend = cli.Command{
	Name:        "send",
	Description: "Send transaction to connected ethereum node with specified arguments",
	Flags: []cli.Flag{
		passphraseFlag,
		passphraseFileFlag,
		keystoreFlag,
		clientFlag,
		senderFlag,
		receiverFlag,
		valueFlag,
		dataFlag,
		syncFlag,
	},
	Action: Send,
}

var commandSendBatch = cli.Command{
	Name:        "sendBatch",
	Description: "Send a batch of transaction to specified ethereum server",
	Flags: []cli.Flag{
		passphraseFlag,
		passphraseFileFlag,
		keystoreFlag,
		clientFlag,
		batchFileFlag,
		batchIndexBeginFlag,
		batchIndexEndFlag,
	},
	Action: SendBatch,
}

// Send sends a transaction with specified fields.
func Send(ctx *cli.Context) error {
	var (
		sender   = ctx.String(senderFlag.Name)
		receiver = ctx.String(receiverFlag.Name)
		value    = ctx.Int(valueFlag.Name)
		data     = ctx.String(dataFlag.Name)
	)
	// Construct call message
	if !CheckArguments(sender, receiver, value, data) {
		return errInvalidArguments
	}
	to := common.HexToAddress(receiver)
	callMsg := &ethereum.CallMsg{
		From:  common.HexToAddress(sender),
		To:    &to,
		Value: big.NewInt(int64(value)),
		Data:  common.FromHex(data),
	}
	// Extract password
	passphrase := getPassphrase(ctx, false)

	// Setup rpc client
	client, err := getClient(ctx)
	if err != nil {
		return err
	}
	keystore := getKeystore(ctx)

	return sendTransaction(client, callMsg, passphrase, keystore, ctx.Bool(syncFlag.Name))
}

// SendBatch sends a batch of specified transactions to ethereum server.
func SendBatch(ctx *cli.Context) error {
	var (
		batchfile = getBatchFile(ctx)
	)
	if _, err := os.Stat(batchfile); os.IsNotExist(err) {
		return err
	}

	var (
		reader Reader
		err    error
		begin  int
		end    int
	)
	switch strings.HasSuffix(batchfile, ".xlsx") {
	case true:
		reader, err = NewExcelReader(batchfile, getSheetId(ctx))
	default:
		reader, err = NewRawTextReader(batchfile)
	}
	if err != nil {
		return err
	}

	entries, err := reader.ReadAll()
	if err != nil {
		return err
	}
	// Read begin, end index for batch file
	begin, end = ctx.Int(batchIndexBeginFlag.Name), ctx.Int(batchIndexEndFlag.Name)
	if end == 0 {
		end = len(entries)
	}

	if begin >= end {
		return errInvalidBatchIndex
	}

	entries = entries[begin:end]

	// Setup rpc client
	client, err := getClient(ctx)
	if err != nil {
		return err
	}
	keystore := getKeystore(ctx)

	for _, entry := range entries {
		// Construct call message
		if !CheckArguments(entry.From.Hex(), entry.To.Hex(), int(entry.Value), common.Bytes2Hex(entry.Data)) {
			return errInvalidArguments
		}
		callMsg := &ethereum.CallMsg{
			From:  entry.From,
			To:    &entry.To,
			Value: big.NewInt(entry.Value),
			Data:  entry.Data,
		}
		if entry.Passphrase == "" {
			entry.Passphrase = getPassphrase(ctx, false)
		}
		// Never wait during the batch sending
		if err := sendTransaction(client, callMsg, entry.Passphrase, keystore, false); err != nil {
			logger.Error(err)
		}
	}
	return nil
}

// sendTransaction sends a transaction with given call message and fill with sufficient fields like account nonce.
func sendTransaction(client *client.Client, callMsg *ethereum.CallMsg, passphrase string, keystore *keystore.KeyStore, wait bool) error {
	gasPrice, gasLimit, nonce, chainId, err := fetchParams(client, callMsg)
	if err != nil {
		return err
	}
	callMsg.Gas = gasLimit
	callMsg.GasPrice = gasPrice
	tx := types.NewTransaction(nonce, *callMsg.To, callMsg.Value, callMsg.Gas, callMsg.GasPrice, callMsg.Data)

	// Sign transaction
	tx, err = keystore.SignTxWithPassphrase(accounts.Account{Address: callMsg.From}, passphrase, tx, chainId)
	if err != nil {
		return err
	}

	// Send transaction
	timeoutContext, _ := makeTimeoutContext(5 * time.Second)
	if err := client.Cli.SendTransaction(timeoutContext, tx); err != nil {
		return err
	}
	logger.Noticef("sendTransaction, hash=%s", tx.Hash().Hex())

	// Wait for the mining
	if wait {
		timeoutContext, _ := makeTimeoutContext(60 * time.Second)
		receipt, err := waitMined(timeoutContext, client, tx.Hash())
		if err != nil {
			logger.Notice("wait transaction receipt failed")
		} else {
			logger.Noticef("transaction receipt=%s", receipt.String())
		}
	}
	return nil
}

// fetchParams returns estimated gas limit, suggested gas price and sender pending nonce.
func fetchParams(client *client.Client, callMsg *ethereum.CallMsg) (*big.Int, uint64, uint64, *big.Int, error) {
	timeoutContext, _ := makeTimeoutContext(5 * time.Second)
	// Gas estimation
	gasLimit, err := client.Cli.EstimateGas(timeoutContext, *callMsg)
	if err != nil {
		return nil, 0, 0, nil, err
	}

	// Suggestion gas price
	timeoutContext, _ = makeTimeoutContext(5 * time.Second)
	gasPrice, err := client.Cli.SuggestGasPrice(timeoutContext)
	if err != nil {
		return nil, 0, 0, nil, err
	}

	// Account Nonce
	timeoutContext, _ = makeTimeoutContext(5 * time.Second)
	nonce, err := client.Cli.PendingNonceAt(timeoutContext, callMsg.From)
	if err != nil {
		return nil, 0, 0, nil, err
	}

	// Chain Id
	timeoutContext, _ = makeTimeoutContext(5 * time.Second)
	chainId, err := client.Cli.NetworkID(timeoutContext)
	if err != nil {
		return nil, 0, 0, nil, err
	}
	// TODO Use cache to improve query efficiency
	return gasPrice, gasLimit, nonce, chainId, nil
}

// waitMined waits the transaction been mined and fetch the receipt.
// An error will been returned if waiting exceeds the given timeout
func waitMined(ctx context.Context, client *client.Client, txHash common.Hash) (*types.Receipt, error) {
	for {
		receipt, err := client.Cli.TransactionReceipt(ctx, txHash)
		if receipt == nil || err != nil {
			time.Sleep(1 * time.Second)
		} else {
			return receipt, nil
		}
		select {
		case <-ctx.Done():
			return nil, errWaitTimeout
		default:
		}
	}
}
