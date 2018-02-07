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
	"io/ioutil"
	"os"
	"path/filepath"

	"errors"
	"github.com/manifoldco/promptui"
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
)

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
