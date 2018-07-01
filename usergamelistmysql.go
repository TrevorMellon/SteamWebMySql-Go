package SteamMySql

import (
	"log"
	"tmp/SteamCommon"

	_ "github.com/go-sql-driver/mysql"
)

func MysqlUpsertSteamUserGames(steamid SteamCommon.SteamID, gamelist []SteamCommon.SteamUserGame) {
	db, _ := getMySqlDB()
	defer db.Close()
	stmt, err := db.Prepare("SELECT COUNT(*) FROM usergames WHERE steamid=? AND appid=?")
	if err != nil {
		log.Println(err)
	}
	stmtinsert, err := db.Prepare("INSERT INTO usergames VALUES(?,?,?)")
	if err != nil {
		log.Println(err)
	}
	stmtupdate, err := db.Prepare("UPDATE usergames SET playtime=? WHERE steamid=? AND appid=?")
	if err != nil {
		log.Println(err)
	}

	defer stmt.Close()
	defer stmtinsert.Close()
	defer stmtupdate.Close()

	exists := make([]bool, len(gamelist))

	for k, v := range gamelist {
		row, _ := stmt.Query(uint64(steamid), v.AppID)
		for row.Next() {
			var c int
			row.Scan(&c)
			if c > 0 {
				exists[k] = true
			}
		}
	}

	tx, _ := db.Begin()
	for k, v := range gamelist {
		if exists[k] {
			stmtupdate.Exec(v.PlaytimeForever, uint64(steamid), v.AppID)
		} else {
			stmtinsert.Exec(uint64(steamid), v.AppID, v.PlaytimeForever)
		}
	}
	tx.Commit()
}
