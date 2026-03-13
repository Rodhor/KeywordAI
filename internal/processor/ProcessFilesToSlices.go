package processor

import (
	"encoding/csv"
	"io"
)

func ParseCSVToSlice(r io.Reader) ([][]string, error) {
	reader := csv.NewReader(r)

	// Uncomment the following line to use a semicolon as the delimiter
	// reader.Comma = ';'

	var data [][]string
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		data = append(data, record)
	}
	return data, nil
}
