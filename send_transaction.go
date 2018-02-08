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
	"errors"
	"math/big"
	"time"
	"context"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/rjl493456442/ethclient/client"
	"gopkg.in/urfave/cli.v1"
)

var (
	errInvalidArguments = errors.New("invalid transaction or call arguments")
	errWaitTimeout = errors.New("wait transaction mined timeout")
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
		keystoreFlag,
		clientFlag,
	},
	Action: SendBatch,
}

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

func SendBatch(ctx *cli.Context) error {
	return nil
}

// sendTransaction send a transaction with given call message and fill with sufficient fields like account nonce.
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

