package kcsv

import (
	"encoding/csv"
	"os"
	"strconv"
)

func LoadCsv(fileName string, hasHeader bool) (map[string][]string, error) {
	f, err := os.Open(fileName)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	r := csv.NewReader(f)
	r.FieldsPerRecord = -1
	records, err := r.ReadAll()
	if err != nil {
		return nil, err
	}
	result := map[string][]string{}
	fieldMap := map[int]string{}
	for i, record := range records {
		if i == 0 {
			if hasHeader {
				for j, field := range record {
					result[field] = []string{}
					fieldMap[j] = field
				}
			} else {
				for j, field := range record {
					result[strconv.Itoa(j)] = []string{field}
					fieldMap[j] = strconv.Itoa(j)
				}
			}
			continue

		}
		for i, field := range record {
			result[fieldMap[i]] = append(result[fieldMap[i]], field)
		}

	}

	return result, nil
}
