package process

import (
	"database/sql"
	"trino-api/internal/model"
)

// this method will parse the column and its type and the rows of data and send it back
func QueryResult(rows *sql.Rows) ([]model.Column, [][]model.Datum, error) {
	var (
		resultColumns []model.Column
		dataRows      [][]model.Datum
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
		var dataRow []model.Datum
		columns := make([]interface{}, len(resultColumns))
		colPtrs := make([]interface{}, len(resultColumns))

		for i := range columns {
			colPtrs[i] = &columns[i]
		}

		if err := rows.Scan(colPtrs...); err != nil {
			return nil, nil, err
		}

		for _, col := range columns {
			dataRow = append(dataRow, model.Datum{Data: col})
		}
		dataRows = append(dataRows, dataRow)
	}
	return resultColumns, dataRows, nil
}
