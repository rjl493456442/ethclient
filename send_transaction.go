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
)

var commandSend = cli.Command{
	Name:        "send",
	Description: "Send transaction to connected ethereum node with specified arguments",
	Flags: []cli.Flag{
		passphraseFlag,
		passphraseFileFlag,
		keystoreFlag,
		clientFlag,
		chainFlag,
		senderFlag,
		receiverFlag,
		valueFlag,
		dataFlag,
	},
	Action: Send,
}

var commandSendBatch = cli.Command{
	Name:        "sendBatch",
	Description: "Send a batch of transaction to specified ethereum server",
	Flags: []cli.Flag{
		keystoreFlag,
		clientFlag,
		chainFlag,
	},
	Action: SendBatch,
}

func Send(ctx *cli.Context) error {
	var (
		sender   = ctx.String(senderFlag.Name)
		receiver = ctx.String(receiverFlag.Name)
		value    = ctx.Int(valueFlag.Name)
		data     = ctx.String(dataFlag.Name)
		chainId  = ctx.Int(chainFlag.Name)
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

	return sendTransaction(client, callMsg, passphrase, keystore, big.NewInt(int64(chainId)), false)
}

func SendBatch(ctx *cli.Context) error {
	return nil
}

// sendTransaction send a transaction with given call message and fill with sufficient fields like account nonce.
func sendTransaction(client *client.Client, callMsg *ethereum.CallMsg, passphrase string, keystore *keystore.KeyStore, chainId *big.Int, wait bool) error {
	gasPrice, gasLimit, nonce, err := fetchParams(client, callMsg)
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

	if wait {
		// TODO Wait the transaction been mined and fetch the receipt.
	}
	logger.Noticef("sendTransaction, hash=%s", tx.Hash().Hex())
	return nil
}

// fetchParams returns estimated gas limit, suggested gas price and sender pending nonce.
func fetchParams(client *client.Client, callMsg *ethereum.CallMsg) (*big.Int, uint64, uint64, error) {
	timeoutContext, _ := makeTimeoutContext(5 * time.Second)
	// Gas estimation
	gasLimit, err := client.Cli.EstimateGas(timeoutContext, *callMsg)
	if err != nil {
		return nil, 0, 0, err
	}

	// Suggestion gas price
	timeoutContext, _ = makeTimeoutContext(5 * time.Second)
	gasPrice, err := client.Cli.SuggestGasPrice(timeoutContext)
	if err != nil {
		return nil, 0, 0, err
	}
	timeoutContext, _ = makeTimeoutContext(5 * time.Second)
	nonce, err := client.Cli.PendingNonceAt(timeoutContext, callMsg.From)
	if err != nil {
		return nil, 0, 0, err
	}
	return gasPrice, gasLimit, nonce, nil
}
