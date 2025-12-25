// Copyright © 2025 chouette.21.00@gmail.com
// Released under the MIT license
// https://opensource.org/licenses/mit-license.php
package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/Chouette2100/srdblib/v2"
)

// StartWorkerPool は指定された数のワーカーGoルーチンを起動します。
func StartWorkerPool(numWorkers int) {
	for i := 0; i < numWorkers; i++ {
		go worker(i)
	}
	log.Printf("%d workers started.", numWorkers)
}

// worker はキューからタスクを取り出し、処理します。
// priorityQueueからの読み込みを優先します。
func worker(id int) {
	for {
		select {
		case task := <-priorityQueue:
			log.Printf("[Worker %d] Processing priority task for room %d", id, task.RoomID)
			processTask(task)
		case task := <-normalQueue:
			log.Printf("[Worker %d] Processing normal task for room %d", id, task.RoomID)
			processTask(task)
		case <-time.After(5 * time.Second): // キューが空の場合のアイドルタイムアウト
			// 必要に応じてワーカーを停止するロジックを追加することも可能
			// log.Printf("[Worker %d] Idling...", id)
		}
	}
}

// processTask は個々のUserTaskを処理します。
// レートリミッターによる待機とaddNewUser関数の呼び出しを行います。
func processTask(task UserTask) {
	// コンテキストがキャンセルされていないかチェック
	select {
	case <-task.Ctx.Done():
		log.Printf("[Worker] Task for room %d cancelled: %v", task.RoomID, task.Ctx.Err())
		if task.IsImmediate {
			task.ErrChan <- task.Ctx.Err()
		}
		return
	default:
		// 続行
	}

	// レートリミッターによる待機
	// task.Ctx を渡すことで、リクエストのタイムアウトやキャンセル時に待機を中断できる
	if err := apiLimiter.Wait(task.Ctx); err != nil {
		log.Printf("[Worker] Rate limiter wait failed for room %d: %v", task.RoomID, err)
		if task.IsImmediate {
			task.ErrChan <- err
		}
		return
	}

	// AddNewUser を呼び出す
	// user, err := AddNewUser(task.RoomID)
	if task.RoomName == "" {
		task.RoomName = fmt.Sprintf("Room_%d", task.RoomID)
	}
	user := &srdblib.User{
		Userno:    task.RoomID,
		User_name: task.RoomName,
	}
	_, err := srdblib.UpinsUser(http.DefaultClient, time.Now(), user)
	if err != nil {
		log.Printf("[Worker] Error processing room %d: %v", task.RoomID, err)
		if task.IsImmediate {
			task.ErrChan <- err
		}
		return
	}

	// 即時処理の場合のみ結果を返す
	if task.IsImmediate {
		select {
		case task.ResultChan <- user:
			// 成功
		case <-task.Ctx.Done():
			log.Printf("[Worker] Result for room %d could not be sent, context cancelled.", task.RoomID)
		case <-time.After(100 * time.Millisecond): // 結果チャネルへの書き込みがブロックされる場合のタイムアウト
			log.Printf("[Worker] Result for room %d could not be sent, receiver not ready.", task.RoomID)
		}
	}
}
