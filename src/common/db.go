package common

import "database/sql"
import "log"
import _ "github.com/Go-SQL-Driver/MySQL"

// openDb opens and tests the database connection.
func OpenDb() (*sql.DB, error) {
	conn, err := sql.Open("mysql",
		"test:@/Convoy?charset=utf8")
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

