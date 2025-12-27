package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func ReadSQL(fn string) (sql string, err error) {
	file, _ := os.Open(fn)
	defer file.Close()

	reader := bufio.NewReader(file)

	// 改行コード '\n' まで読み込む
	line, err := reader.ReadString('\n')
	if err != nil {
		err = fmt.Errorf("Failed to read SQL from file %s: %v", fn, err)
		return "", err
	}

	// 改行を除去したい場合
	line = strings.TrimRight(line, "\n\r")
	return line, nil
}
