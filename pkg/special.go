package pkg

import (
	"database/sql"
	"log"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

func getChromeHistory(path string, fn string) int {
	if fn == "" {
		fn = "History"
	}

	pathHistory := filepath.Join(filepath.Dir(path), fn)

	ver := getSqlite(pathHistory, "SELECT values FROM meta WHERE key='version'", nil)[0]
	if ver <= 1 {
		panic("ver must be greater than 1")
	}

	return ver
}

func shredSliteCharColumns(table string, cols int, where string) {
	cmd := ""
	if cols > 0 && 
}

type sqlDt int

func getSqlite(path string, command string, parameters interface{}) []int {
	ids := []int{}

	db, err := sql.Open("sqlite3", path)
	if err != nil {
		log.Fatalln(err.Error())
	}
	defer db.Close()

	rows, err := db.Query(command, parameters)
	if err != nil {
		log.Fatalln(err.Error())
	}
	defer rows.Close()

	for rows.Next() {
		dt := new(sqlDt)
		rows.Scan(dt)
		ids = append(ids, int(*dt))
	}

	return ids
}
