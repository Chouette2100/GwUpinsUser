// Copyright © 2025 chouette.21.00@gmail.com
// Released under the MIT license
// https://opensource.org/licenses/mit-license.php
package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-gorp/gorp"
	_ "github.com/go-sql-driver/mysql" // MySQLドライバー

	"github.com/Chouette2100/srapi/v2"
	"github.com/Chouette2100/srcom"
	"github.com/Chouette2100/srdblib/v2"
)

/*
0000000 最初のバージョン
0000100 CheckEqIrankZero.go, CheckOldData.go を追加する
*/

var Version = "0000100"

// GwUpinsUser: A Go application to add new users via HTTP requests with worker pool and rate limiting.
func main() {
	// ログファイルの作成
	logfile, err := srcom.CreateLogfile3(Version, srapi.Version, srdblib.Version)
	if err != nil {
		log.Printf("ログファイルの作成に失敗しました。%v\n", err)
		return
	}
	defer logfile.Close()

	// DB接続
	var dbconfig *srdblib.DBConfig
	dbconfig, err = srdblib.OpenDb("DBConfig.yml")
	if err != nil {
		log.Printf("Database error. err = %v\n", err)
		return
	}
	if dbconfig.UseSSH {
		defer srdblib.Dialer.Close()
	}
	defer srdblib.Db.Close()
	srdblib.Db.SetMaxOpenConns(8)
	srdblib.Db.SetMaxIdleConns(12)

	srdblib.Db.SetConnMaxLifetime(time.Minute * 5)
	srdblib.Db.SetConnMaxIdleTime(time.Minute * 5)

	defer srdblib.Db.Close()
	log.Printf("%+v\n", dbconfig)

	dial := gorp.MySQLDialect{Engine: "InnoDB", Encoding: "utf8mb4"}
	srdblib.Dbmap = &gorp.DbMap{Db: srdblib.Db,
		Dialect:         dial,
		ExpandSliceArgs: true, //スライス引数展開オプションを有効化する
	}
	// --------------------------------

	// DB接続設定
	srdblib.Db.SetMaxOpenConns(20)                 // 最大オープン接続数
	srdblib.Db.SetMaxIdleConns(10)                 // アイドル接続数
	srdblib.Db.SetConnMaxLifetime(5 * time.Minute) // 接続の最大再利用時間

	// DbMap = &gorp.DbMap{Db: db, Dialect: gorp.MySQLDialect{Engine: "InnoDB", Encoding: "UTF8"}}

	// テーブルマッピング
	// DbMap.AddTableWithName(User{}, "users").SetKeys(true, "ID")

	// 開発時のみ: テーブルが存在しない場合に作成
	// 本番環境ではマイグレーションツールなどを使用することを推奨
	// err = DbMap.CreateTablesIfNotExists()
	// if err != nil {
	// 	log.Fatalf("Failed to create tables: %v", err)
	// }

	srdblib.AddTableWithName()

	log.Println("Database initialized successfully.")

	// ワーカープールを起動 (例: 5つのワーカー)
	StartWorkerPool(5)

	go CheckEqIrankZero()
	go CheckOldData()

	// HTTPハンドラの設定
	http.HandleFunc("/AddNewUser", AddNewUserHandler)
	http.HandleFunc("/AddNewUserImmediately", AddNewUserImmediatelyHandler)

	// HTTPサーバーの起動
	// ポート番号を環境変数から取得する、指定がなければデフォルトで8080を使用
	port := ":8080"
	if p := os.Getenv("PORT"); p != "" {
		port = ":" + p
	}
	server := &http.Server{
		Addr:         port,
		Handler:      nil, // デフォルトのMuxを使用
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 35 * time.Second, // /AddNewUserImmediately の処理時間に合わせて調整
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("Server starting on port %s", port)
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Could not listen on %s: %v\n", port, err)
		}
	}()

	// シャットダウン処理
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Server shutting down...")

	// サーバーのGraceful Shutdown
	// context.WithTimeout を使ってシャットダウンのタイムアウトを設定
	// ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	// defer cancel()
	// if err := server.Shutdown(ctx); err != nil {
	// 	log.Fatalf("Server forced to shutdown: %v", err)
	// }

	log.Println("Server exited gracefully.")
}
