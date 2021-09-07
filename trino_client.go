package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/trinodb/trino-go-client/trino"
)

func main() {
	dsn := "https://user@trino-dev-coordinator.de.razorpay.com:443?catalog=default&schema=test"
	db, err := sql.Open("trino", dsn)

	if err != nil {
		log.Fatal(err)
	}
	db.Query("SELECT 1", 1, sql.Named("X-Trino-User", string("Alice")))
	fmt.Println("wtf")

}
