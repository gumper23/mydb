package main

import (
	"bufio"
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	host := "127.0.0.1"
	port := "13306"
	schema := "information_schema"
	username, password, err := ReadMyCnf()

	dsn := username + ":" + password + "@tcp(" + host + ":" + port + ")/" + schema
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err = db.Ping(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	tables, err := GetRows(db, "select concat(table_schema, '.', table_name) as tbl from information_schema.tables")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	for _, table := range tables {
		tbl := table["tbl"]
		fmt.Println(tbl)
		results, err := GetRows(db, "select * from "+tbl+" limit 10")
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}

		for _, result := range results {
			PrintRow(result)
		}
		fmt.Println()
	}
}

func GetRows(db *sql.DB, query string) (results []map[string]string, err error) {
	rows, err := db.Query(query)
	if err != nil {
		return
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return
	}

	rawResult := make([][]byte, len(cols))
	dest := make([]interface{}, len(cols))
	for i := range rawResult {
		dest[i] = &rawResult[i]
	}

	for rows.Next() {
		row := make(map[string]string, len(cols))
		err = rows.Scan(dest...)
		if err != nil {
			return
		}

		for i, raw := range rawResult {
			if raw == nil {
				row[cols[i]] = "NULL"
			} else {
				row[cols[i]] = string(raw)
			}
		}
		results = append(results, row)
	}
	err = rows.Err()
	return
}

func GetRow(db *sql.DB, query string) (row map[string]string, err error) {
	rows, err := GetRows(db, query)
	if err != nil {
		return
	} else if len(rows) == 0 {
		err = sql.ErrNoRows
		return
	} else {
		return rows[0], nil
	}
}

func PrintRow(row map[string]string) {
	for key, value := range row {
		fmt.Printf("[%s] = [%s]\n", key, value)
	}
}

func ReadMyCnf() (username, password string, err error) {
	usr, err := user.Current()
	if err != nil {
		return
	}

	file, err := os.Open(filepath.Join(usr.HomeDir, ".my.cnf"))
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if err = scanner.Err(); err != nil {
		return
	}
	for scanner.Scan() {
		s := scanner.Text()
		s = strings.ReplaceAll(s, " ", "")
		fmt.Println(s)

		if strings.Contains(s, "=") {
			elems := strings.Split(s, "=")
			if elems[0] == "user" {
				username = elems[1]
			} else if elems[0] == "password" {
				password = elems[1]
			}
			if len(username) > 0 && len(password) > 0 {
				break
			}
		}
	}
	return
}
