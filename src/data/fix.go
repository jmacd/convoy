package data

import "database/sql"
import "log"

import "common"

func FixCityNames(db *sql.DB, table, column string) error {
	query := "SELECT DISTINCT (" + column + ") COLLATE utf8_bin FROM " + table;
	rows, err := db.Query(query)
	if err != nil {
		return err
	}
	defer rows.Close()
	type Fix struct {
		city, correct string
	}
	var fixes []Fix
	for rows.Next() {
		var cityb []byte
		if err := rows.Scan(&cityb); err != nil {
			return err
		}
		city := string(cityb)
		correct := common.ProperName(city)
		if correct == city {
			continue
		}
		fixes = append(fixes, Fix{city, correct})
	}
	if err := rows.Err(); err != nil {
		return err
	}
	stmt, err := db.Prepare("UPDATE " + table + " SET " +
		column + " = ? WHERE " + column + " LIKE BINARY ?")
	if err != nil {
		log.Fatal("Could not prepare UPDATE statement")
	}

	for _, fix := range fixes {
		_, err := stmt.Exec(fix.correct, fix.city)
		if err != nil {
			return err
		}
		log.Println("Replaced", fix.city, "with", fix.correct)
	}
	return nil
}

