package utils

import (
	"errors"

	"github.com/tidwall/gjson"
)

var (
	ErrGjsonNotFound  = errors.New("specified path does not exist")
	ErrGjsonWrongType = errors.New("wrong type")
)

func GjsonGet(json []byte, path string) (gjson.Result, error) {
	result := gjson.GetBytes(json, path)
	if !result.Exists() {
		return result, ErrGjsonNotFound
	}

	return result, nil
}

func GjsonParseStringMap(jsonObject string) (map[string]string, error) {
	if jsonObject == "" {
		return nil, nil
	}

	result := gjson.Parse(jsonObject)
	if !result.IsObject() {
		return nil, ErrGjsonWrongType
	}

	ret := make(map[string]string)
	result.ForEach(func(key, value gjson.Result) bool {
		ret[key.String()] = value.String()
		return true
	})

	return ret, nil
}
