// Copyright © 2025 chouette.21.00@gmail.com
// Released under the MIT license
// https://opensource.org/licenses/mit-license.php
package main

import (
	"context"
	"log"
	"time"

	"github.com/Chouette2100/srdblib/v2" // srdblibパッケージをインポート
	"golang.org/x/time/rate"
)

// UserTask はワーカーが処理するタスクの情報を保持します。
type UserTask struct {
	RoomID      int
	RoomName    string
	IsImmediate bool               // trueなら即時処理、falseなら通常処理
	ResultChan  chan *srdblib.User // Immediateの場合のみ結果を返すチャネル
	ErrChan     chan error         // Immediateの場合のみエラーを返すチャネル
	Ctx         context.Context    // リクエストのコンテキスト (タイムアウトやキャンセル用)
}

var (
	normalQueue   chan UserTask
	priorityQueue chan UserTask
	apiLimiter    *rate.Limiter // 外部API呼び出しのレートリミッター
)

func init() {
	// キューの初期化 (バッファサイズはシステムの負荷に応じて調整)
	normalQueue = make(chan UserTask, 1000)
	priorityQueue = make(chan UserTask, 100) // 優先キューは通常キューより小さくても良い

	// レートリミッターの初期化
	// 例: 1秒あたり5回のリクエストを許可し、バーストは10まで
	// rate.Every(time.Second/5) は 200ms に1トークン生成 (1秒に5トークン)
	// apiLimiter = rate.NewLimiter(rate.Every(time.Second/5), 10)
	apiLimiter = rate.NewLimiter(rate.Every(time.Second*6), 1)
	log.Printf("Rate limiter initialized: %v tokens/sec, burst %d", apiLimiter.Limit(), apiLimiter.Burst())
}
