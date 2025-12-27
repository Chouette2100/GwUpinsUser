package main

import (
	"context"
	"log"
	"time"

	_ "github.com/go-sql-driver/mysql" // MySQLドライバー

	"github.com/Chouette2100/srdblib/v2"
)

// irankが0のユーザーをノーマルキューに追加する
func CheckEqIrankZero() {
	var err error
	log.Printf("CheckEqIrankZero started.")
	ticker := time.NewTicker(19 * time.Minute)
	defer ticker.Stop()

	// {関数名}.txt（CheckEqIrankZero.txt）からSQL文を読み込む
	var sqlst string

	for {
		sqlst, err = ReadSQL("CheckEqIrankZero.txt")
		log.Printf("SQL: %s", sqlst)

		var users []srdblib.User
		// sqlst := "SELECT * FROM user WHERE irank = 0 ORDER BY ts LIMIT 10"
		_, err = srdblib.Dbmap.Select(&users, sqlst)
		if err != nil {
			log.Printf("Database select error in CheckEqIrankZero: %v", err)
		}

		if len(users) == 0 {
			log.Printf("No users with irank=0 found.")
		} else {
			ticker_i := time.NewTicker(15 * time.Second)
			// defer ticker_i.Stop()
			for _, user := range users {
				task := UserTask{
					RoomID:      user.Userno,
					RoomName:    user.User_name,
					IsImmediate: false,
					Ctx:         context.Background(),
				}

				// タスクを通常キューに投入
				select {
				case normalQueue <- task:
					log.Printf("[CheckEqIrankZero] Room %d queued.", user.Userno)
				case <-time.After(50 * time.Millisecond): // キューが満杯の場合のタイムアウト
					log.Printf("[CheckEqIrankZero] Normal queue full, failed to queue room %d.", user.Userno)
				}
				<-ticker_i.C
			}
			ticker_i.Stop()
		}

		// 次のティックを待つ
		<-ticker.C
	}
}
