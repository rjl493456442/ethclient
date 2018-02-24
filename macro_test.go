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
	"os"
	"path"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/rjl493456442/ethclient/client"
)

func TestReadTokenList(t *testing.T) {
	tokens, err := ReadTokenList(path.Join("test", "ethToken.json"))
	if err != nil {
		t.Error(err)
	}
	if len(tokens) == 0 {
		t.Error("emty token list read from file")
	}

	tokens, err = ReadTokenList("")
	if err != nil {
		t.Error(err)
	}
	if len(tokens) == 0 {
		t.Error("emty token list fetch from github")
	}
	os.Remove(ethTokenFile)
}

func TestParse(t *testing.T) {
	var (
		cmdBalanceOf = "#BALANCEOF RDN 0x8f0909ccb296ebd319834edb0d5785794b781d7f"
		cmdTransfer  = "#TRANSFER RDN 100"
	)
	cli, err := client.NewClient("http://172.16.5.3:9999")
	if err != nil {
		t.Error(err)
	}
	parser, err := NewMacroParser(cli, path.Join("test", "rinkebyEthToken.json"))
	if err != nil {
		t.Error(err)
	}
	addr, payload, decimal, err := parser.Parse(cmdBalanceOf, "", "")
	if err != nil {
		t.Error(err)
	}
	if !checkEqual(addr, common.HexToAddress("0xe10f51424adbead82eb4b9ae72c29828dc24188f"),
		payload, "70a082310000000000000000000000008f0909ccb296ebd319834edb0d5785794b781d7f", decimal, 18) {
		t.Error("invalid parse result")
	}

	addr, payload, decimal, err = parser.Parse(cmdTransfer, "0xadd0354d4f5c101685509001053730417321db49", "0x8f0909ccb296ebd319834edb0d5785794b781d7f")
	if err != nil {
		t.Error(err)
	}
	// Ignore decimal returned here.
	if !checkEqual(addr, common.HexToAddress("0xe10f51424adbead82eb4b9ae72c29828dc24188f"),
		payload, "a9059cbb0000000000000000000000008f0909ccb296ebd319834edb0d5785794b781d7f0000000000000000000000000000000000000000000000056bc75e2d63100000",
		decimal, 0) {
		t.Error("invalid parse result")
	}
}

func checkEqual(addr, addrExpect common.Address, payload, payloadExpect string, decimal, decimalExpect int) bool {
	if strings.ToLower(addr.Hex()) != strings.ToLower(addrExpect.Hex()) {
		return false
	}
	if decimal != decimalExpect {
		return false
	}
	if payload != payloadExpect {
		return false
	}
	return true
}
