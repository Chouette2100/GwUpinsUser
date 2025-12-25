// Copyright © 2025 chouette.21.00@gmail.com
// Released under the MIT license
// https://opensource.org/licenses/mit-license.php
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/Chouette2100/srdblib/v2" // srdblibパッケージをインポート
)

// AddNewUserHandler は /AddNewUser エンドポイントを処理します。
// リクエストを受け取ったらすぐにレスポンスを返し、処理は非同期で行われます。
func AddNewUserHandler(w http.ResponseWriter, r *http.Request) {
	roomIDStr := r.URL.Query().Get("roomid")
	roomID, err := strconv.Atoi(roomIDStr)
	if err != nil {
		http.Error(w, "Invalid roomid", http.StatusBadRequest)
		return
	}
	roomName := r.URL.Query().Get("roomname")
	if roomName == "" {
		roomName = fmt.Sprintf("Room_%d", roomID)
	}

	// 仮データをuserテーブルに格納する
	user := &srdblib.User{
		Userno:    roomID,
		User_name: roomName,
		Longname:  roomName,
		Irank:     888888888,
		Ts:        time.Date(2000, 1, 1, 0, 0, 0, 0, time.Local),
	}
	err = srdblib.Dbmap.Insert(user)
	if err != nil {
		log.Printf("Database insert error for room %d: %v", roomID, err)
	}

	task := UserTask{
		RoomID:      roomID,
		RoomName:    roomName,
		IsImmediate: false,
		// Ctx:         r.Context(), // リクエストのコンテキストをワーカーに渡す
		// ここを修正: HTTPリクエストのコンテキストではなく、
		// アプリケーションのバックグラウンドコンテキストを使用する
		Ctx: context.Background(),
	}

	// タスクを通常キューに投入
	select {
	case normalQueue <- task:
		w.WriteHeader(http.StatusAccepted) // 202 Accepted を返す
		fmt.Fprintf(w, "Task for room %d queued successfully for asynchronous processing.", roomID)
		log.Printf("[Handler /AddNewUser] Room %d queued.", roomID)
	case <-time.After(50 * time.Millisecond): // キューが満杯の場合のタイムアウト
		http.Error(w, "Service busy, please try again later.", http.StatusServiceUnavailable)
		log.Printf("[Handler /AddNewUser] Room %d queue full, returning 503.", roomID)
	}
}

// AddNewUserImmediatelyHandler は /AddNewUserImmediately エンドポイントを処理します。
// リクエストを受け取ったら、処理が完了するまで待機し、結果をJSONで返します。
func AddNewUserImmediatelyHandler(w http.ResponseWriter, r *http.Request) {
	roomIDStr := r.URL.Query().Get("roomid")
	roomID, err := strconv.Atoi(roomIDStr)
	if err != nil {
		http.Error(w, "Invalid roomid", http.StatusBadRequest)
		return
	}

	resultChan := make(chan *srdblib.User, 1) // 結果を待つためのチャネル
	errChan := make(chan error, 1)            // エラーを待つためのチャネル

	task := UserTask{
		RoomID:      roomID,
		IsImmediate: true,
		ResultChan:  resultChan,
		ErrChan:     errChan,
		Ctx:         r.Context(), // リクエストのコンテキストをワーカーに渡す
	}

	// タスクを優先キューに投入
	select {
	case priorityQueue <- task:
		log.Printf("[Handler /AddNewUserImmediately] Room %d queued as priority. Waiting for result...", roomID)
		// キューにタスクを投入後、結果を待つ
		select {
		case user := <-resultChan:
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(user)
			log.Printf("[Handler /AddNewUserImmediately] Room %d processed successfully, returned JSON.", roomID)
		case err := <-errChan:
			http.Error(w, fmt.Sprintf("Failed to process immediately: %v", err), http.StatusInternalServerError)
			log.Printf("[Handler /AddNewUserImmediately] Room %d failed with error: %v", roomID, err)
		case <-r.Context().Done(): // クライアントが接続を切った場合など
			http.Error(w, "Request cancelled by client.", http.StatusGatewayTimeout)
			log.Printf("[Handler /AddNewUserImmediately] Room %d request cancelled by client.", roomID)
		case <-time.After(30 * time.Second): // 処理のタイムアウト (適宜調整)
			http.Error(w, "Processing timed out.", http.StatusGatewayTimeout)
			log.Printf("[Handler /AddNewUserImmediately] Room %d processing timed out.", roomID)
		}
	case <-time.After(50 * time.Millisecond): // キューが満杯の場合のタイムアウト
		http.Error(w, "Service busy for immediate tasks, please try again later.", http.StatusServiceUnavailable)
		log.Printf("[Handler /AddNewUserImmediately] Room %d priority queue full, returning 503.", roomID)
	}
}
