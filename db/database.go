package database

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"flag"

	"github.com/jackc/pgx"
	_"github.com/lib/pq"
)

type DBTypeError string

type DBError struct {
	Type  string
	Model string
}

func (dberr *DBError) Error() string {
	return dberr.Type + dberr.Model
}

const (
	ERRORALREADYEXISTS = " already exists "
	ERROR_DONT_EXISTS  = " don't exists "
)

var DB *pgx.ConnPool

func InitDB() (err error) {
	var port = flag.Int("p", 5432, "help message for flagname")
	var db = flag.String("db", "forum_db", "blabla")

	//port, _ := strconv.Atoi(os.Getenv("PG_PORT"))
	//db := os.Getenv("PG_DB")
	config := pgx.ConnConfig{
		Host:     "localhost",
		User:     "docker",
		Password: "docker",
		Database: *db,
		Port:     uint16(*port),
		//RuntimeParams: map[string]string{
		//	"sslmode": "disable",
		//},
	}

	DB, err = pgx.NewConnPool(
		pgx.ConnPoolConfig{
			ConnConfig:     config,
			MaxConnections: 50,
		})

	if err != nil {
		log.Fatal(err)
	}
	//DBH, err = fastsql.Open("postgres", fmt.Sprintf("user=docker host=localhost password=docker dbname=forum_db port=%s sslmode=disable", port), 50)
	//log.Println(err)
	initFile, err := ioutil.ReadFile(fmt.Sprintf("%s/src/github.com/zwirec/tech-db/init.sql", os.Getenv("GOPATH")))
	if err != nil {
		log.Fatal(err)
	}
	tx, _ := DB.Begin()

	_, err = tx.Exec(string(initFile))

	if err != nil {
		tx.Rollback()
		log.Println(err)
	}
	tx.Commit()
	DB.Reset()
	return nil
}
