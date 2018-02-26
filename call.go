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
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/rjl493456442/ethclient/client"
	"gopkg.in/urfave/cli.v1"
)

var commandCall = cli.Command{
	Name:        "call",
	Usage:       "Execute a message call transaction in the remote node's VM",
	Description: "Call ethereum smart contract in the connected remote node without leaving trace on the blockchain",
	Flags: []cli.Flag{
		clientFlag,
		senderFlag,
		receiverFlag,
		valueFlag,
		dataFlag,
	},
	Action: Call,
}

func Call(ctx *cli.Context) error {
	var (
		sender   = ctx.String(senderFlag.Name)
		receiver = ctx.String(receiverFlag.Name)
		value    = ctx.Int(valueFlag.Name)
		data     = ctx.String(dataFlag.Name)
	)
	// Construct call message
	if !CheckArguments(sender, receiver, value, common.FromHex(data)) {
		return errInvalidArguments
	}
	to := common.HexToAddress(receiver)
	callMsg := &ethereum.CallMsg{
		From:  common.HexToAddress(sender),
		To:    &to,
		Value: big.NewInt(int64(value)),
		Data:  common.FromHex(data),
	}

	// Setup rpc client
	client, err := getClient(ctx)
	if err != nil {
		return err
	}

	result, err := call(client, callMsg)
	if err != nil {
		logger.Error(err)
	} else {
		logger.Noticef("Result=%s", common.Bytes2Hex(result))
	}
	return nil
}

func call(client *client.Client, callMsg *ethereum.CallMsg) ([]byte, error) {
	ctx, _ := makeTimeoutContext(5 * time.Second)
	return client.Cli.CallContract(ctx, *callMsg, nil)
}
