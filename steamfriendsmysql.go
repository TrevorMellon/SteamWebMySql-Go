package SteamMySql

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"time"
	"tmp/SteamCommon"
	"unicode/utf8"

	_ "github.com/go-sql-driver/mysql"
)

var DBSettings DatabaseSettings

func MysqlCheckUser(steamid SteamCommon.SteamID) (bool, SteamCommon.SteamUserProfile) {
	id := make([]uint64, 1)
	id[0] = uint64(steamid)
	table, b := MysqlUserProfilesFromIds(id)
	return b[0], table[0]
}

func getMySqlDB() (*sql.DB, error) {
	dsn := DBSettings.Username + ":"
	dsn += DBSettings.Password + "@"
	dsn += "tcp("
	dsn += DBSettings.Host
	dsn += ":" + strconv.FormatUint(uint64(DBSettings.Port), 10)
	dsn += ")"
	dsn += "/" + DBSettings.Name
	dsn += "?" + "parseTime=true"
	dsn += "&collation=utf8mb4_general_ci"
	dsn += "&charset=utf8mb4,utf8"
	db, err := sql.Open("mysql", dsn)
	return db, err
}

func MysqlUserProfilesFromIds(ids []uint64) ([]SteamCommon.SteamUserProfile, []bool) {
	q := "SELECT personaname, realname, personanameblob, realnameblob, "
	q += " url, dateadded, avsmall, avmedium, avlarge"
	q += " FROM user WHERE steamid=?"
	db, _ := getMySqlDB()
	defer db.Close()
	stmt, err := db.Prepare(q)
	if err != nil {
		log.Println(err)
	}
	tables := make([]SteamCommon.SteamUserProfile, len(ids))
	exists := make([]bool, len(ids))

	checkstmt, _ := db.Prepare("SELECT COUNT(*) FROM user WHERE steamid=?")
	if err != nil {
		log.Println(err)
	}
	for k, v := range ids {
		rows, _ := checkstmt.Query(v)
		for rows.Next() {
			var c int
			err := rows.Scan(&c)
			if err != nil {
				log.Println(err)
			}
			if c > 0 {
				exists[k] = true
			}
		}
	}

	for k, v := range ids {
		if exists[k] {
			t := &tables[k]
			t.SteamID = SteamCommon.SteamID(v)
			rows, err := stmt.Query(v)
			if err != nil {
				log.Println(err)
			}
			for rows.Next() {
				err := rows.Scan(&t.PersonaName, &t.Realname,
					&t.DisplayPersona, &t.DisplayRealname, &t.Url, &t.DateAdded,
					&t.Avatar.Small, &t.Avatar.Medium, &t.Avatar.Large)
				if err != nil {
					log.Println(err)
				}
				name := t.PersonaName

				if b := utf8.ValidString(name); !b {
					r := []rune(name)
					name = ""
					for _, v := range r {
						name += strconv.QuoteRuneToASCII(v)
					}
				}
				t.PersonaName = name

				real := t.Realname

				if b := utf8.ValidString(real); !b {
					r := []rune(real)
					real = ""
					for _, v := range r {
						real += strconv.QuoteRuneToASCII(v)
					}
				}
				t.Realname = real
			}
			stmt.Close()
			q = "SELECT summaryparsed FROM user WHERE steamid=?"
			stmt, _ = db.Prepare(q)
			rows, _ = stmt.Query(v)
			var a time.Time
			for rows.Next() {
				err = rows.Scan(&a)
				if err == nil {
					t.SummaryParsed = a
				}
			}
		}
	}
	return tables, exists
}

func MysqlUpsertFriends(user SteamCommon.SteamUserProfile) {
	steamid := user.SteamID
	db, err := getMySqlDB()
	if err != nil {
		log.Println(err)
	}
	defer db.Close()
	stmt, _ := db.Prepare("SELECT COUNT(*) FROM friends WHERE steamid=? AND friendid=?")
	insertstmt, _ := db.Prepare("INSERT INTO friends VALUES(?,?,?,?)")

	exists := make([]bool, len(user.Friends))

	for k, v := range user.Friends {
		rows, _ := stmt.Query(steamid, v.SteamID)
		for rows.Next() {
			var c int
			rows.Scan(&c)
			if c > 0 {
				exists[k] = true
			}
		}
	}

	tx, _ := db.Begin()

	for k, v := range user.Friends {
		if !exists[k] {
			t := v.FriendSince
			//t := time.Unix(tt, 0)

			_, err := insertstmt.Exec(
				steamid,
				v.SteamID,
				t,
				time.Now())

			if err != nil {
				log.Println(err)
			}
		}
		//r.Close()
	}
	tx.Commit()
	insertstmt.Close()
	stmt.Close()

	stmt, _ = db.Prepare("SELECT COUNT(*) FROM user WHERE steamid=?")

	iq := "INSERT INTO user (steamid, personaname, realname, personanameblob, realnameblob, url, dateadded, "
	iq += "avsmall, avmedium, avlarge, profiletype, status"
	iq += ") VALUES(?,?,?,?,?,?,?,?,?,?,?,?)"

	insertstmt, err = db.Prepare(iq)
	if err != nil {
		log.Println(err)
	}

	tx, _ = db.Begin()

	exists = make([]bool, len(user.Friends))

	for k, v := range user.Friends {
		rows, _ := stmt.Query(v.SteamID)
		for rows.Next() {
			var c int
			rows.Scan(&c)
			if c > 0 {
				exists[k] = true
			}
		}
	}

	for k, v := range user.Friends {
		name := v.DisplayPersona
		if b := utf8.ValidString(name); !b {
			r := []rune(name)
			name = ""
			for _, v := range r {
				name += strconv.QuoteRuneToASCII(v)
			}

		}
		realname := v.DisplayRealname
		if b := utf8.ValidString(realname); !b {
			r := []rune(realname)
			realname = ""
			for _, v := range r {
				realname = strconv.QuoteRuneToASCII(v)
			}

		}

		if exists[k] {
			q := "UPDATE user SET personaname=?, realname=?, personanameblob=?, realnameblob=?, url=?, "
			q += "avsmall=?, avmedium=?, avlarge=?, profiletype=?, status=? WHERE steamid=?"

			_, err := db.Exec(q,
				name,
				realname,
				v.PersonaName,
				v.DisplayRealname,
				v.Url,
				v.Avatar.Small,
				v.Avatar.Medium,
				v.Avatar.Large,
				v.ProfileType,
				v.Status,
				v.SteamID)

			if err != nil {
				fmt.Println(err)
			}
		} else {
			insertstmt.Exec(
				v.SteamID,
				name,
				realname,
				v.PersonaName,
				v.Realname,
				v.Url,
				time.Now(),
				v.Avatar.Small,
				v.Avatar.Medium,
				v.Avatar.Large,
				v.ProfileType,
				v.Status)
		}

		/*b, err := res.RowsAffected()
		if b > int64(0) {
			log.Println(res.RowsAffected)
		}
		if err != nil {
			log.Println(err)
		}*/
	}
	tx.Commit()
	stmt.Close()
	insertstmt.Close()
	//updatestmt.Close()
}

func MysqlGetFriends(user *SteamCommon.SteamUserProfile) {
	steamid := uint64(user.SteamID)

	db, _ := getMySqlDB()
	defer db.Close()

	stmt, err := db.Prepare("SELECT friendid, friendssince FROM friends WHERE steamid=?")
	if err != nil {
		log.Println(err)
	}
	countstmt, err := db.Prepare("SELECT COUNT(*) from friends WHERE steamid=?")
	if err != nil {
		log.Println(err)
	}
	q := "SELECT personaname, realname, personanameblob, realnameblob, "
	q += "url, dateadded, avsmall, avmedium, avlarge, profiletype, status FROM user WHERE steamid=?"

	userstmt, err := db.Prepare(q)
	if err != nil {
		log.Println(err)
	}

	rows, err := countstmt.Query(steamid)

	if err != nil {
		fmt.Println(err)
	}

	var count int
	for rows.Next() {
		rows.Scan(&count)
	}

	if count > 0 {

		//friends := make([]SteamFriendsSince.FriendSince, count)
		user.Friends = make([]SteamCommon.SteamUserProfile, count)
		var idx int

		rows, err := stmt.Query(steamid)
		if err != nil {
			log.Println(err)
		}
		for rows.Next() {
			var id uint64
			var t time.Time
			rows.Scan(&id,
				&t)

			user.Friends[idx].FriendSince = t
			user.Friends[idx].SteamID = SteamCommon.SteamID(id)
			log.Println(t)

			idx++
		}

		for k, v := range user.Friends {

			rows, err := userstmt.Query(v.SteamID)

			var s SteamCommon.SteamUserProfile

			if err != nil {
				log.Println(err)
			}

			id := uint64(user.Friends[k].SteamID)

			for rows.Next() {
				err = rows.Scan(&s.PersonaName, &s.Realname,
					&s.DisplayPersona, &s.DisplayRealname,
					&s.Url, &s.DateAdded,
					&s.Avatar.Small, &s.Avatar.Medium, &s.Avatar.Large,
					&s.ProfileType, &s.Status)
				if err != nil {
					log.Println(err)
				}
			}
			t := user.Friends[k].FriendSince
			s.FriendSince = t
			s.SteamID = SteamCommon.SteamID(id)
			if s.DisplayPersona == "" {
				s.DisplayPersona = s.PersonaName
			}
			if s.DisplayRealname == "" {
				s.DisplayRealname = s.Realname
			}
			user.Friends[k] = s
			//log.Println(s)
		}

	}
	//return nil
}

func MySqlUpsertUser(profile SteamCommon.SteamUserProfile) {
	db, _ := getMySqlDB()
	defer db.Close()
	countstmt, err := db.Prepare("SELECT COUNT(*) FROM user WHERE steamid=?")
	if err != nil {
		log.Println(err)
	}
	rows, err := countstmt.Query(uint64(profile.SteamID))
	if err != nil {
		log.Println(err)
	}
	var exists bool
	for rows.Next() {
		var c int
		rows.Scan(&c)
		if c > 0 {
			exists = true
		}
	}
	countstmt.Close()

	if exists {
		q := "UPDATE user SET personaname=?, realname=?, "
		q += "personanameblob=?, realnameblob=?, "
		q += "url=?,  avsmall=?, "
		q += "avmedium=?, avlarge=?, summaryparsed=?"
		q += " WHERE steamid=?"

		_, err := db.Exec(q,
			profile.PersonaName, profile.Realname,
			profile.DisplayPersona, profile.DisplayRealname,
			profile.Url,
			profile.Avatar.Small, profile.Avatar.Medium,
			profile.Avatar.Large, time.Now(),
			uint64(profile.SteamID))
		if err != nil {
			log.Println(err)
		}

	} else {
		q := "INSERT INTO user VALUES(?,?,?,?,?,?,?,?,?,?,?)"
		_, err = db.Exec(q, uint64(profile.SteamID), profile.PersonaName, profile.Realname,
			profile.DisplayPersona, profile.DisplayRealname,
			profile.Url,
			time.Now(), time.Now(),
			profile.Avatar.Small, profile.Avatar.Medium,
			profile.Avatar.Large)
		if err != nil {
			log.Println(err)
		}

	}
}
