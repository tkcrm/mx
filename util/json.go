package util

import (
	"github.com/goccy/go-json"
)

func JSONToStruct[T any](src any) (T, error) {
	var res T
	result, err := json.Marshal(src)
	if err != nil {
		return res, err
	}

	if err := json.Unmarshal(result, &res); err != nil {
		return res, err
	}

	return res, nil
}

func BytesToStruct[T any](data []byte) (T, error) {
	var res T
	if err := json.Unmarshal(data, &res); err != nil {
		return res, err
	}
	return res, nil
}
