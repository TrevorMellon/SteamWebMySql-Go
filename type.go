package SteamMySql

type DatabaseType int

const (
	DatabaseNone DatabaseType = iota
	DatabaseMySQL
	DatabaseSqlite
)

type DatabaseSettings struct {
	Type     DatabaseType
	Name     string
	Host     string
	Port     int
	Username string
	Password string
}
