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
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/360EntSecGroup-Skylar/excelize"
	"github.com/ethereum/go-ethereum/common"
)

var (
	errEmptycanner      = errors.New("empty scanner")
	errInvalidContent   = errors.New("invalid file content")
	errEmptyFileContent = errors.New("empty file content")
)

// ErrCorrupted describes error due to corruption. This error will be wrapped
// with errors.ErrCorrupted.
type ErrCorrupted struct {
	Pos    int64
	Size   int64
	Kind   string
	Reason string
}

func (e *ErrCorrupted) Error() string {
	return fmt.Sprintf("file: corruption on %s (pos=%d): %s", e.Kind, e.Pos, e.Reason)
}

// TransactionParams packages transaction related fields.
type TransactionParams struct {
	From       common.Address `json:"from"`
	To         common.Address `json:"to"`
	Value      int64          `json:"value"`
	Data       []byte         `json:"data"`
	Passphrase string         `json:"passphrase"`
}

type Reader interface {
	Read() (TransactionParams, error)
	ReadAll() ([]TransactionParams, error)
}

/*
	Json Reader
*/

/*
	Excel Reader
*/

const DefaultSheet = "Sheet1"

type ExcelReader struct {
	fd    *excelize.File
	sheet string
	idx   int
}

func NewExcelReader(filename string, sheet string) (Reader, error) {
	fd, err := excelize.OpenFile(filename)
	if err != nil {
		return nil, err
	}
	return &ExcelReader{
		fd:    fd,
		sheet: sheet,
		idx:   0,
	}, nil
}

func (reader *ExcelReader) Read() (TransactionParams, error) {
	rows := reader.fd.GetRows(reader.sheet)
	if len(rows) < 1 {
		return TransactionParams{}, errEmptyFileContent
	}
	reader.idx += 1
	if len(rows) >= reader.idx {
		return TransactionParams{}, io.EOF
	}
	row := rows[reader.idx]
	return reader.parseRow(row, reader.idx)
}

func (reader *ExcelReader) ReadAll() ([]TransactionParams, error) {
	rows := reader.fd.GetRows(reader.sheet)
	if len(rows) < 1 {
		return nil, errEmptyFileContent
	}
	var params []TransactionParams
	for idx, row := range rows[1:] {
		if param, err := reader.parseRow(row, idx); err == nil {
			params = append(params, param)
		}
	}
	return params, nil
}

func (reader *ExcelReader) parseRow(row []string, idx int) (TransactionParams, error) {
	if len(row) != 5 {
		return TransactionParams{}, errInvalidContent
	}
	for i := 0; i < 5; i++ {
		// Remove all leading and trailing blank char
		row[i] = strings.Trim(row[i], " ")
	}
	value, err := strconv.ParseInt(row[2], 10, 64)
	if err != nil {
		logger.Errorf("Corrupted raw text line at %d, invalid transfer value %s", idx, value)
		return TransactionParams{}, err
	}
	return TransactionParams{
		From:       common.HexToAddress(row[0]),
		To:         common.HexToAddress(row[1]),
		Value:      value,
		Data:       common.FromHex(row[3]),
		Passphrase: row[4],
	}, nil
}

/*
	Raw Text Reader
*/

// RTReader a reader to read raw text file.
// Note, raw text file line format:
// <sender>, <receiver>, <value>, <payload>, <passphrase>
type RawTextReader struct {
	fd      *os.File
	scanner *bufio.Scanner
}

func NewRawTextReader(filename string) (Reader, error) {
	fd, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	scanner := bufio.NewScanner(fd)
	return &RawTextReader{
		fd:      fd,
		scanner: scanner,
	}, nil
}

// ReadAll read single line in raw text reader and parse it in fixed format.
func (reader *RawTextReader) Read() (TransactionParams, error) {
	if reader.scanner == nil {
		return TransactionParams{}, errEmptycanner
	}
	if reader.scanner.Scan() {
		return reader.parseLine(reader.scanner.Text(), 0)
	} else {
		if err := reader.scanner.Err(); err != nil {
			return TransactionParams{}, err
		} else {
			return TransactionParams{}, io.EOF
		}
	}
}

// ReadAll reads all content in raw text reader and parse all lines in fixed format.
func (reader *RawTextReader) ReadAll() ([]TransactionParams, error) {
	if reader.scanner == nil {
		return nil, errEmptycanner
	}
	var (
		idx    = 0
		params = []TransactionParams{}
	)
	for reader.scanner.Scan() {
		p, err := reader.parseLine(reader.scanner.Text(), idx)
		if err == nil {
			params = append(params, p)
		}
		idx += 1
	}
	if err := reader.scanner.Err(); err != nil {
		return nil, err
	}
	return params, nil
}

func (reader *RawTextReader) parseLine(line string, idx int) (TransactionParams, error) {
	substr := strings.Split(line, ",")
	if len(substr) != 5 {
		logger.Errorf("Corrupted raw text line at %d", idx)
		return TransactionParams{}, errInvalidContent
	}
	for i := 0; i < 5; i++ {
		// Remove all leading and trailing blank char
		substr[i] = strings.Trim(substr[i], " ")
	}
	value, err := strconv.ParseInt(substr[2], 10, 64)
	if err != nil {
		logger.Errorf("Corrupted raw text line at %d, invalid transfer value %s", idx, value)
		return TransactionParams{}, err
	}
	return TransactionParams{
		From:       common.HexToAddress(substr[0]),
		To:         common.HexToAddress(substr[1]),
		Value:      value,
		Data:       common.FromHex(substr[3]),
		Passphrase: substr[4],
	}, nil
}
