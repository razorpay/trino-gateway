package process

import (
	"database/sql"

	"github.com/razorpay/trino-gateway/internal/trino_rest/model"
)

type QueryProcessor interface {
	QueryResult(rows *sql.Rows) ([]model.Column, []map[string]interface{}, error)
}

type DefaultProcessor struct{}

// this method will parse the column and its type and the rows of data and send it back
func (p *DefaultProcessor) QueryResult(rows *sql.Rows) ([]model.Column, []map[string]interface{}, error) {
	var (
		resultColumns []model.Column
		dataRows      []map[string]interface{}
	)
	columns, err := rows.Columns()
	if err != nil {
		return nil, nil, err
	}

	colTypes, err := rows.ColumnTypes()
	if err != nil {
		return nil, nil, err
	}

	for i, col := range columns {
		resultColumns = append(resultColumns, model.Column{
			Name: col,
			Type: colTypes[i].DatabaseTypeName(),
		})
	}

	for rows.Next() {
		columns := make([]interface{}, len(resultColumns))
		colPtrs := make([]interface{}, len(resultColumns))
		rowMap := make(map[string]interface{})

		for i := range columns {
			colPtrs[i] = &columns[i]
		}

		if err := rows.Scan(colPtrs...); err != nil {
			return nil, nil, err
		}

		for i, col := range columns {
			rowMap[resultColumns[i].Name] = col
		}
		dataRows = append(dataRows, rowMap)
	}
	return resultColumns, dataRows, nil
}
