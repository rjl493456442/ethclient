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
	"crypto/ecdsa"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/pborman/uuid"
	"gopkg.in/urfave/cli.v1"
	"path"
)

var commandGenerate = cli.Command{
	Name:        "generate",
	Usage:       "generate new keyfile",
	Description: "Generate one or a batch of new keyfile.",
	Flags: []cli.Flag{
		passphraseFlag,
		passphraseFileFlag,
		keystoreFlag,
		cli.IntFlag{
			Name:  "number",
			Usage: "required account number to generate",
			Value: 1,
		},
	},
	Action: func(ctx *cli.Context) error {
		var (
			privateKey *ecdsa.PrivateKey
			prefix     string
			err        error
			number     int
		)

		prefix = ctx.String(keystoreFlag.Name)
		// Create keystore directory if not exist
		if _, err := os.Stat(prefix); os.IsNotExist(err) {
			os.MkdirAll(prefix, 0700)
		}

		accountList, err := os.OpenFile(path.Join(prefix, "addresses"), os.O_APPEND|os.O_CREATE|os.O_RDWR, 0777)
		if err != nil {
			fmt.Println("open addresses file failed")
			return err
		}
		number = ctx.Int("number")
		if number > 1 {
			fmt.Printf("Generate %d ethereum account required\n", number)
		}

		for i := 0; i < number; i++ {
			privateKey, err = crypto.GenerateKey()
			if err != nil {
				fmt.Println("Failed to generate random private key", err)
			}
			// Create the keyfile object with a random UUID.
			id := uuid.NewRandom()
			key := &keystore.Key{
				Id:         id,
				Address:    crypto.PubkeyToAddress(privateKey.PublicKey),
				PrivateKey: privateKey,
			}

			// Encrypt key with passphrase.
			passphrase := getPassphrase(ctx, true)
			keyjson, err := keystore.EncryptKey(key, passphrase, keystore.StandardScryptN, keystore.StandardScryptP)
			if err != nil {
				fmt.Println("Error encrypting key", err)
			}

			keyfilepath := keystore.KeyFileName(key.Address)
			keyfilepath = path.Join(prefix, keyfilepath)
			// Store the file to disk.
			if err := os.MkdirAll(filepath.Dir(keyfilepath), 0700); err != nil {
				fmt.Printf("Could not create directory %s\n", filepath.Dir(keyfilepath))
			}
			if err := ioutil.WriteFile(keyfilepath, keyjson, 0600); err != nil {
				fmt.Printf("Failed to write keyfile to %s: %v\n", keyfilepath, err)
			}

			// Output some information.
			fmt.Println("Address:", key.Address.Hex())
			accountList.WriteString(key.Address.Hex() + "\n")
		}
		return nil
	},
}
