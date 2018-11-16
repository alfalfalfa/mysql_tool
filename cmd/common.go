package cmd

import (
	"encoding/json"
	"fmt"
	"strconv"
)

func checkError(err error) {
	if err != nil {
		panic(err)
	}
}

func e(err error) {
	if err != nil {
		panic(err.Error())
	}
}

func s(v interface{}) string {
	if v == nil {
		return ""
	}
	return fmt.Sprint(v)
}
func b(v interface{}) bool {
	if v == nil {
		return false
	}
	if bb, ok := v.(bool); ok {
		return bb
	}
	return false
}

func i(v interface{}) int {
	if v == nil {
		return 0
	}
	if ii, ok := v.(int); ok {
		return ii
	}
	if sv, ok := v.(string); ok {
		ii, err := strconv.Atoi(sv)
		if err != nil {
			return 0
		}
		return ii
	}
	return 0
}

func dump(v interface{}) {
	j, err := json.MarshalIndent(v, "", "\t")
	if err != nil {
		panic(err)
	}
	fmt.Println(string(j))
	//fmt.Sprintln(string(j))
}
