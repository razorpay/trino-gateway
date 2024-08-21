package trino

import (
	"database/sql"
	"time"

	// blank import as this is needed to implement trino go client

	_ "github.com/trinodb/trino-go-client/trino"
)

type Client struct {
	db *sql.DB
}

type TrinoClient interface {
	Query(query string) (*sql.Rows, error)
}

func NewTrinoClient(dsn string) (*Client, error) {
	db, err := sql.Open("trino", dsn)
	if err != nil {
		return nil, err
	}

	db.SetConnMaxLifetime(1 * time.Minute)
	return &Client{db: db}, nil
}
func (client *Client) Close() error {
	return client.db.Close()
}

func (client *Client) Query(query string) (*sql.Rows, error) {
	return client.db.Query(query)
}
