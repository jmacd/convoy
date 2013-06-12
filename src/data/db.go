package data

import "database/sql"
import "flag"
import "log"
import "strings"
import "runtime"
import _ "github.com/Go-SQL-Driver/MySQL"

import "common"

type TableName string

var dbName = flag.String("db_name", "", "Name of the DB")

// OpenDb opens and tests the database connection.
func OpenDb() (*sql.DB, error) {
	if len(*dbName) == 0 {
		log.Fatal("Database not specified, use --db_name")
	}
	conn, err := sql.Open("mysql",
		"test:@/"+*dbName+"?charset=utf8")
	if err != nil {
		return conn, err
	}
	// Test that the connection is good; because the driver call
	// to open the database is defered until the first request.
	_, err = conn.Exec("SELECT 1;")
	if err != nil {
		log.Fatal("Database not opened!", err)
	}
	return conn, err
}

func Table(s TableName) string {
	return *dbName + "." + string(s)
}

func insertPlaceHolders(columns []string) string {
	qs := make([]string, len(columns))
	for i, _ := range qs {
		qs[i] = "?"
	}
	return strings.Join(qs, ", ")
}

func wherePlaceHolders(columns []string) string {
	parts := make([]string, len(columns))
	for i, col := range columns {
		parts[i] = col + " = ?"
	}
	return strings.Join(parts, " AND ")
}

func InsertQuery(db *sql.DB, table TableName, columns ...string) (*sql.Stmt, error) {
	return db.Prepare("INSERT INTO " +
		Table(table) + " (" + strings.Join(columns, ", ") + ") VALUES (" +
		insertPlaceHolders(columns) + ")")
}

func SelectWhereQuery(db *sql.DB, table TableName, columns ...string) (*sql.Stmt, error) {
	return db.Prepare("SELECT * FROM " + Table(table) +
		" WHERE " + wherePlaceHolders(columns))
}

func SelectGroupQuery(db *sql.DB, table TableName, columns ...string) (*sql.Stmt, error) {
	cols := strings.Join(columns, ", ")
	return db.Prepare("SELECT " + cols +
		" FROM " + Table(table) +
		" GROUP BY " + cols)
}

func SelectAllQuery(db *sql.DB, table TableName, columns ...string) (*sql.Stmt, error) {
	cols := strings.Join(columns, ", ")
	return db.Prepare("SELECT " + cols + " FROM " + Table(table))
}

func HasRows(s *sql.Stmt, a ...interface{}) (bool, error) {
	has, err := s.Query(a...)
	if err != nil {
		return false, err
	}
	defer has.Close()
	if has.Next() {
		return true, nil
	}
	if err := has.Err(); err != nil {
		return false, err
	}
	return false, nil
}

func ForAll(stmt *sql.Stmt, afunc func() error, a ...interface{}) error {
	rows, err := stmt.Query()
	if err != nil {
		return err
	}

	defer rows.Close()
	for rows.Next() {
		if err := rows.Scan(a...); err != nil {
			return err
		}
		if err := afunc(); err != nil {
			return err
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	return nil
}

func Main(body func (*sql.DB) error) {
	flag.Parse()
	argv := flag.Args()
	runtime.GOMAXPROCS(common.NumCPU())
	if len(argv) != 0 {
		log.Fatalln("Extra args:", argv)
	}
	db, err := OpenDb()
	if err != nil {
		log.Fatal("Could not open database", err)
	}
	defer db.Close()
	
	if err := body(db); err != nil {
		log.Fatal("Program error", err)
	}
}
