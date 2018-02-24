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

// Smart contract invocation data is also allowed to be represented as macro format.
// e.g. #TRANSFER EOS 2000 means 2000 eos token will transfer from sender to receiver.
//
// With marco definition, user can customize invocation data easily but with limitation of macro types.
// Current support macro definitions:
// #TRANSFER  <token symbol> <token number>|<token percentage>
// #BALANCEOF <token symbol> <address>
package main

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"math"
	"math/big"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/rjl493456442/ethclient/client"
	"github.com/rjl493456442/ethclient/resource"
)

const (
	ethTokenJsonURL = "https://raw.githubusercontent.com/kvhnuke/etherwallet/mercury/app/scripts/tokens/ethTokens.json"
	ethTokenFile    = "ethToken.json"
)

const (
	MacroTransfer  = "transfer"
	MacroBalanceOf = "balanceof"
)

var (
	errNoDownloadToolInstalled   = errors.New("no download tool installed")
	errInvalidMacroDefinition    = errors.New("invalid macro definition")
	errInvalidMacroArgument      = errors.New("invalid macro argument")
	errUndefinedMacro            = errors.New("undefined macro definition")
	errUnrecognizableTokenSymbol = errors.New("the given token symbol is unrecognizable")
)

var (
	macroSet = NewMacroSet()
)

// Token packages all fields of a ECR20 token
type Token struct {
	Address string `json:"address"`
	Symbol  string `json:"symbol"`
	Decimal int    `json:"decimal"`
	Type    string `json:"type"`
}

// ReadTokenList reads all available token information from the given file or fetch from the share json file in github.
func ReadTokenList(path string) ([]Token, error) {
	var (
		tokens  []Token
		content []byte
		err     error
	)
	// If token list file is not been specified, fetch token list from the github
	if _, err := os.Stat(path); err != nil && os.IsNotExist(err) {
		content, err = downloadTokenList()
		if err != nil {
			return nil, err
		}
	} else {
		content, err = ioutil.ReadFile(path)
		if err != nil {
			return nil, err
		}
	}
	err = json.Unmarshal(content, &tokens)
	if err != nil {
		return nil, err
	}
	return tokens, nil
}

// downloadTokenList downloads token list from the given url.
func downloadTokenList() ([]byte, error) {
	var (
		toolType    int = 0
		downloadCmd *exec.Cmd
	)
	// Make sure the download tool has been installed
	isWgetInstalled := func() bool {
		cmd := exec.Command("wget", "--help")
		if err := cmd.Run(); err != nil {
			return false
		}
		downloadCmd = exec.Command("wget", "-O", ethTokenFile, ethTokenJsonURL)
		return true
	}
	isCurlInstalled := func() bool {
		cmd := exec.Command("curl", "--help")
		if err := cmd.Run(); err != nil {
			return false
		}
		downloadCmd = exec.Command("curl", "-o", ethTokenFile, ethTokenJsonURL)
		return true
	}
	if isWgetInstalled() {
		toolType = 1
	} else if isCurlInstalled() {
		toolType = 2
	}

	if toolType == 0 {
		return nil, errNoDownloadToolInstalled
	}

	if err := downloadCmd.Run(); err != nil {
		return nil, err
	}

	return ioutil.ReadFile(ethTokenFile)
}

type Macro struct {
	ArgNumber int
}

func NewMacroSet() map[string]Macro {
	return map[string]Macro{
		MacroTransfer: {
			ArgNumber: 2,
		},
		MacroBalanceOf: {
			ArgNumber: 2,
		},
	}
}

type MacroParser struct {
	client *client.Client
	tokens map[string]Token
}

func NewMacroParser(client *client.Client, path string) (*MacroParser, error) {
	tokenList, err := ReadTokenList(path)
	if err != nil {
		return nil, err
	}

	tokens := map[string]Token{}
	for _, token := range tokenList {
		tokens[strings.ToLower(token.Symbol)] = token
	}

	parser := &MacroParser{
		client: client,
		tokens: tokens,
	}
	return parser, nil
}

// Parse parses the given macro definition string and returns a valid contract invocation data.
func (mp *MacroParser) Parse(input, sender, receiver string) (common.Address, string, int, error) {
	lines := strings.Split(input, " ")
	for idx, line := range lines {
		lines[idx] = strings.Trim(line, " ")
	}

	if len(lines) < 1 {
		return common.Address{}, "", 0, errInvalidMacroDefinition
	}

	keyword := lines[0]
	if !strings.HasPrefix(keyword, "#") {
		return common.Address{}, "", 0, errInvalidMacroDefinition
	}
	switch strings.ToLower(keyword[1:]) {
	case MacroTransfer:
		addr, payload, err := mp.parseTransfer(lines[1:], sender, receiver)
		return addr, payload, 0, err
	case MacroBalanceOf:
		return mp.parseBalanceOf(lines[1:])
	default:
		return common.Address{}, "", 0, errUndefinedMacro
	}
}

// parseTransfer parses transfer macro.
// Transfer macro syntax:
// #TRANSFER <Token symbol> (<Token number> Or <Token percentage>)
// Return value:
// contract address, invocation data and error
func (mp *MacroParser) parseTransfer(lines []string, sender, receiver string) (common.Address, string, error) {
	var (
		address string
		decimal int
		amount  *big.Int
		err     error
	)
	if len(lines) != macroSet[MacroTransfer].ArgNumber {
		return common.Address{}, "", errInvalidMacroArgument
	}
	symbol := lines[0]
	if token, exist := mp.tokens[strings.ToLower(symbol)]; exist {
		address = token.Address
		decimal = token.Decimal
	} else {
		return common.Address{}, "", errUnrecognizableTokenSymbol
	}

	if strings.HasSuffix(lines[1], "%") {
		_, err := strconv.ParseFloat(lines[1][:len(lines[1])-1], 64)
		if err != nil {
			return common.Address{}, "", err
		}
		// Fetch the balance
		// TODO
		amount = big.NewInt(0)
	} else {
		v, err := strconv.ParseInt(lines[1], 10, 64)
		if err != nil {
			return common.Address{}, "", err
		}
		amount = big.NewInt(v)
		amount.Mul(amount, big.NewInt(int64(math.Pow10(decimal))))
	}
	// Assemble the payload
	parsed, err := abi.JSON(strings.NewReader(resource.ERC20InterfaceABI))
	if err != nil {
		return common.Address{}, "", err
	}
	input, err := parsed.Pack("transfer", common.HexToAddress(receiver), amount)
	if err != nil {
		return common.Address{}, "", err
	}
	return common.HexToAddress(address), common.Bytes2Hex(input), err
}

// parseTransfer parses balanceOf macro.
// BalanceOf macro syntax:
// #BALANCEOF <Token symbol> <token holder address>
// Return value:
// contract address, invocation data, decimal and error
func (mp *MacroParser) parseBalanceOf(lines []string) (common.Address, string, int, error) {
	var (
		address string
		decimal int
		err     error
	)
	if len(lines) != macroSet[MacroBalanceOf].ArgNumber {
		return common.Address{}, "", 0, errInvalidMacroArgument
	}
	symbol := lines[0]
	if token, exist := mp.tokens[strings.ToLower(symbol)]; exist {
		address = token.Address
		decimal = token.Decimal
	} else {
		return common.Address{}, "", 0, errUnrecognizableTokenSymbol
	}
	// Assemble the payload
	parsed, err := abi.JSON(strings.NewReader(resource.ERC20InterfaceABI))
	if err != nil {
		return common.Address{}, "", 0, err
	}
	input, err := parsed.Pack("balanceOf", common.HexToAddress(lines[1]))
	if err != nil {
		return common.Address{}, "", 0, err
	}
	return common.HexToAddress(address), common.Bytes2Hex(input), decimal, err
}
