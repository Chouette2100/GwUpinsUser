package main

import (
	"context"
	"log"
	"time"

	_ "github.com/go-sql-driver/mysql" // MySQLドライバー

	"github.com/Chouette2100/srdblib/v2"
)

// irankが0のユーザーをノーマルキューに追加する
func CheckOldData() {
	var err error
	log.Printf("CheckOldData started.")
	ticker := time.NewTicker(9 * time.Minute)
	defer ticker.Stop()

	var sqlst string

	for {
		var users []srdblib.User

		sqlst, err = ReadSQL("CheckOldData.txt")
		log.Printf("SQL: %s", sqlst)

		// sqlst := `
		// 	SELECT *
		//	    FROM user
		//		WHERE ts BETWEEN '2010-01-01' AND  NOW()-INTERVAL 1 MONTH
		//		ORDER BY ts LIMIT 5
		//`
		_, err = srdblib.Dbmap.Select(&users, sqlst)
		if err != nil {
			log.Printf("Database select error in CheckOldData: %v", err)
		}

		if len(users) == 0 {
			log.Printf("No old user data found.")
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
					log.Printf("[CheckOldData] Room %d queued.", user.Userno)
				case <-time.After(50 * time.Millisecond): // キューが満杯の場合のタイムアウト
					log.Printf("[CheckOldData] Normal queue full, failed to queue room %d.", user.Userno)
				}
				<-ticker_i.C
			}
			ticker_i.Stop()
		}

		// 次のティックを待つ
		<-ticker.C
	}
}
