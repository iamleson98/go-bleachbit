package pkg

import (
	"database/sql"
	"fmt"
	"log"
	"path/filepath"
	"strings"

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

// Create an SQL command to shred character columns
func shredSqliteCharColumns(table string, cols []string, where string) string {
	cmd := ""
	if len(cols) > 0 && options_.get("shred", "", getBool) == true {
		stringList1 := []string{}
		stringList2 := []string{}
		for _, col := range cols {
			stringList1 = append(stringList1, fmt.Sprintf("%s = randomblob(length(%s))", col, col))
			stringList2 = append(stringList2, fmt.Sprintf("%s = zeroblob(length(%s))", col, col))
		}
		cmd += fmt.Sprintf("update or ignore %s set %s %s;", table, strings.Join(stringList1, ","), where)
		cmd += fmt.Sprintf("update or ignore %s set %s %s;", table, strings.Join(stringList2, ","), where)
	}

	cmd += fmt.Sprintf("delete from %s %s;", table, where)
	return cmd
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
