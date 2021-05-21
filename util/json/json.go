package json

import (
	"bytes"
	"encoding/json"

	"github.com/alfalfalfa/mysql_tool/util/errors"
	"log"
)

func ToJson(el interface{}) string {
	b, err := json.Marshal(el)
	if err != nil {
		log.Fatal(errors.Wrap(err))
	}

	b, err = prettyprint(b)
	if err != nil {
		log.Fatal(errors.Wrap(err))
	}
	return string(b)
}
func Json(el interface{}, err error) (string, error) {
	if el == nil || err != nil {
		return "", err
	}
	b, err := json.Marshal(el)
	if err != nil {
		log.Fatal(errors.Wrap(err))
	}
	return string(b), err
}

func prettyprint(b []byte) ([]byte, error) {
	var out bytes.Buffer
	err := json.Indent(&out, b, "", "  ")
	return out.Bytes(), err
}
