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
	"fmt"
	"path"
)

func ExampleRTReaderRead() {
	reader, err := NewRawTextReader(path.Join("test", "raw_text"))
	if err != nil {
		return
	}
	for i := 0; i < 3; i++ {
		param, err := reader.Read()
		if err != nil {
			return
		}
		fmt.Println(param.From.Hex())
	}
	// Output:
	// 0x7236Bc5a9Ff647D48b1eceaa07aa6438dCca615e
	// 0x7Cd6342b4b02A90bcf60F1f843d1002897e38b1f
	// 0xfFc1736f670f305A3d752280d07F6895379cbD70
}

func ExampleRTReaderReadAll() {
	reader, err := NewRawTextReader(path.Join("test", "raw_text"))
	if err != nil {
		return
	}
	params, err := reader.ReadAll()
	if err != nil {
		return
	}
	for _, param := range params {
		fmt.Println(param.From.Hex())
	}
	// Output:
	// 0x7236Bc5a9Ff647D48b1eceaa07aa6438dCca615e
	// 0x7Cd6342b4b02A90bcf60F1f843d1002897e38b1f
	// 0xfFc1736f670f305A3d752280d07F6895379cbD70
}

func ExampleExcelReaderRead() {
	reader, err := NewRawTextReader(path.Join("test", "raw_text"))
	if err != nil {
		return
	}
	for i := 0; i < 3; i++ {
		param, err := reader.Read()
		if err != nil {
			return
		}
		fmt.Println(param.From.Hex())
	}
	// Output:
	// 0x7236Bc5a9Ff647D48b1eceaa07aa6438dCca615e
	// 0x7Cd6342b4b02A90bcf60F1f843d1002897e38b1f
	// 0xfFc1736f670f305A3d752280d07F6895379cbD70
}

func ExampleExcelReaderReadAll() {
	reader, err := NewExcelReader(path.Join("test", "excel.xlsx"), DefaultSheet)
	if err != nil {
		return
	}
	params, err := reader.ReadAll()
	if err != nil {
		return
	}
	for _, param := range params {
		fmt.Println(param.From.Hex())
	}
	// Output:
	// 0x7236Bc5a9Ff647D48b1eceaa07aa6438dCca615e
	// 0x7Cd6342b4b02A90bcf60F1f843d1002897e38b1f
	// 0xfFc1736f670f305A3d752280d07F6895379cbD70
}
