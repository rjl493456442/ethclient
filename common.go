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
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/manifoldco/promptui"
	"github.com/rjl493456442/ethclient/client"
	"gopkg.in/urfave/cli.v1"
)

var (
	passphraseFileFlag = cli.StringFlag{
		Name:  "passwordfile",
		Usage: "the file that contains the passphrase for the keyfile",
	}
	passphraseFlag = cli.StringFlag{
		Name:  "password",
		Usage: "keyfile associated passphrase",
	}
	keystoreFlag = cli.StringFlag{
		Name:  "keystore",
		Usage: "keystore directory path",
		Value: "keystore",
	}
	clientFlag = cli.StringFlag{
		Name:  "url",
		Usage: "remote ethereum server url, http/https/ws/ipc are all supported",
	}
	senderFlag = cli.StringFlag{
		Name:  "sender",
		Usage: "transaction sender address",
	}
	receiverFlag = cli.StringFlag{
		Name:  "receiver",
		Usage: "transaction receiver address",
	}
	valueFlag = cli.IntFlag{
		Name:  "value",
		Usage: "transfer value(wei)",
	}
	dataFlag = cli.StringFlag{
		Name:  "data",
		Usage: "contract invocation payload",
	}
	syncFlag = cli.BoolFlag{
		Name:  "sync",
		Usage: "wait until the sending transaction been mined",
	}
	batchFileFlag = cli.StringFlag{
		Name:  "batchfile",
		Usage: "file path for batch transaction sending defail",
	}
	batchIndexBeginFlag = cli.IntFlag{
		Name:  "batchstart",
		Usage: "beginning index for batch transaction sending. If not specifed, the default value is 0",
		Value: 0,
	}
	batchIndexEndFlag = cli.IntFlag{
		Name:  "batchend",
		Usage: "end index for batch transaction sending. If not specified, the default value is len(entries) - 1",
		Value: 0,
	}
	sheetFlag = cli.StringFlag{
		Name:  "sheet",
		Usage: "excel file sheet id",
	}
	tokenfileFlag = cli.StringFlag{
		Name:  "tokenfile",
		Usage: "customized token file path which in json format",
	}
)

// CheckArguments make sure the arguments assigned are valid.
func CheckArguments(sender, receiver string, value int, payload []byte) bool {
	if strings.HasPrefix(sender, "0x") {
		sender = sender[2:]
	}
	if strings.HasPrefix(receiver, "0x") {
		receiver = receiver[2:]
	}
	if sender == "" || len(sender) != 40 {
		return false
	}
	if receiver != "" && len(receiver) != 40 {
		return false
	}
	if receiver == "" && len(payload) == 0 {
		return false
	}
	if value < 0 {
		return false
	}
	return true
}

// createCommandLineApp returns an application instance with sufficient fields.
func createCommandLineApp() *cli.App {
	app := cli.NewApp()
	app.Name = filepath.Base(os.Args[0])
	app.Author = "Gary rong"
	app.Usage = "ethclient is an easy ethereum client, with which user can interactive with ethereum client easily"
	app.Email = "garyrong0905@gmail.com"
	app.Version = "1.0.0-alpha"
	return app
}

// getPassphrase fetches keyfile passphrase from command option.
func getPassphrase(ctx *cli.Context, confirmation bool) string {
	var (
		passphrase        string
		confirmPassphrase string
		err               error
	)
	// Get passphrase from passphrase flag. Note, it is not recommended out out security problem.
	if passphrase = ctx.String(passphraseFlag.Name); passphrase != "" {
		return passphrase
	}
	// Get passphrase from passphraseFile flag.
	if fname := ctx.String(passphraseFileFlag.Name); fname != "" {
		content, err := ioutil.ReadFile(fname)
		if err == nil {
			passphrase = string(content)
			return passphrase
		}
	}

	validate := func(input string) error {
		if len(input) < 6 {
			return errors.New("Password must have more than 6 characters")
		}
		return nil
	}
	prompt := promptui.Prompt{
		Label:    "Passphrase",
		Validate: validate,
		Mask:     '*',
	}
	// Get passphrase from command line prompt
	for {
		passphrase, err = prompt.Run()
		if err == nil {
			break
		}
	}
	if confirmation {
		prompt.Label = "Confirmation"
		confirmPassphrase, err = prompt.Run()
		if err != nil || confirmPassphrase != passphrase {
			return ""
		}
	}
	return passphrase
}

// getClient returns a remote client connected to specified ethereum server.
func getClient(ctx *cli.Context) (*client.Client, error) {
	url := ctx.String(clientFlag.Name)
	return client.NewClient(url)
}

// getKeystore returns a keystore with given file path.
func getKeystore(ctx *cli.Context) *keystore.KeyStore {
	path := ctx.String(keystoreFlag.Name)
	keystore := keystore.NewKeyStore(path, keystore.StandardScryptN, keystore.StandardScryptP)
	return keystore
}

// getBatchFile extracts batch file path from command line input or console input.
func getBatchFile(ctx *cli.Context) string {
	path := ctx.String(batchFileFlag.Name)
	if path == "" {
		// Read from the console input
		prompt := promptui.Prompt{
			Label: "Batchfile path",
		}
		path, _ = prompt.Run()
	}
	return path
}

// getMacroParser returns a macro definition parser.
func getMacroParser(client *client.Client, path string) (*MacroParser, error) {
	return NewMacroParser(client, path)
}

// getSheetId returns excel sheet id from command line input.
// If no specified, use the default sheet id.
func getSheetId(ctx *cli.Context) string {
	sheet := ctx.String(sheetFlag.Name)
	if sheet == "" {
		sheet = DefaultSheet
	}
	return sheet
}

// makeContext returns background context.
func makeContext() context.Context {
	return context.Background()
}

// makeTimeoutContext returns timeout context with given expire duration.
func makeTimeoutContext(timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(makeContext(), timeout)
}
