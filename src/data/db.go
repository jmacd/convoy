package data

import "database/sql"
import "flag"
import "log"
import _ "github.com/Go-SQL-Driver/MySQL"

const (
	Corrections = "Corrections"
	Locations = "Locations"
	TruckLoads = "TruckLoads"
	LoadCityStates = "LoadCityStates"
	GeoCityStates = "GeoCityStates"
	UnknownLocations = "UnknownLocations"
)

var dbName = flag.String("db_name", "", "Name of the DB")

// OpenDb opens and tests the database connection.
func OpenDb() (*sql.DB, error) {
	if len(*dbName) == 0 {
		log.Fatal("Database not specified, use --db_name")
	}
	conn, err := sql.Open("mysql",
		"test:@/" + *dbName + "?charset=utf8")
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

func Table(s string) string {
	return *dbName + "." + s
}