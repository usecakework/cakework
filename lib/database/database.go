package database

// import (
// 	"database/sql"
// 	"fmt"
// 	"time"

// 	_ "github.com/go-sql-driver/mysql"
// 	log "github.com/sirupsen/logrus"
// )

// type Database struct {
// 	Db *sql.DB
// }

// func New(dsn string) (*Database, error) {
// 	db, err := sql.Open("mysql", dsn)
// 	if err != nil {
// 		return nil, err
// 	}
// 	db.SetConnMaxLifetime(time.Minute * 3)
// 	db.SetMaxOpenConns(100)
// 	db.SetMaxIdleConns(10)

// 	database := &Database{
// 		Db: db,
// 	}

// 	// defer db.Close() // does this statement work here once this function ends?

// 	return database, nil
// }

// func (database *Database) GetOneResult(query string, target string, result any) error {
// 	fmt.Println(query)
// 	err := database.Db.QueryRow(query, target).Scan(&result)

// 	if err != nil {
// 		if err.Error() == sql.ErrNoRows.Error() {
// 			log.Info("No results matching query: " + query + " and target: " + target + " found")
// 		}
// 		return err
// 	} else {
// 		return nil
// 	}
// }