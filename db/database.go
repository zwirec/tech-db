package database

import (
	"github.com/jmoiron/sqlx"
	_"github.com/lib/pq"
	"log"
	"io/ioutil"
	"os"
	"fmt"
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
	ERROR_ALREADY_EXISTS = " already exists "
	ERROR_DONT_EXISTS    = " don't exists "
)

var DB *sqlx.DB
//var DBH *fastsql.DB

func InitDB() (err error) {
	port := os.Getenv("PG_PORT")
	DB, err = sqlx.Connect("postgres", fmt.Sprintf("user=docker host=localhost password=docker dbname=forum_db port=%s sslmode=disable", port))
	if err != nil {
		log.Fatal(err)
	}
	//DBH, err = fastsql.Open("postgres", fmt.Sprintf("user=docker host=localhost password=docker dbname=forum_db port=%s sslmode=disable", port), 100)
	DB.SetMaxIdleConns(100)
	DB.SetMaxOpenConns(100)
	DB.SetConnMaxLifetime(0)
	initFile, err := ioutil.ReadFile(fmt.Sprintf("%s/src/github.com/zwirec/tech-db/init.sql", os.Getenv("GOPATH")))
	if err != nil {
		log.Fatal(err)
	}
	_, err = DB.Exec(string(initFile))
	return nil
}
