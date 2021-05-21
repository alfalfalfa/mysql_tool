/*
動的変数コピー
コンパイラチェックなし、直書きのほうが早いので要注意
*/
package copy

import (
	"fmt"
	"reflect"
	"strconv"

	"github.com/alfalfalfa/mysql_tool/util/errors"
)

//TODO interface{} == ポインタのチェック, 変換
func copy(v1 reflect.Value, v2 reflect.Value) bool {
	if !v1.IsValid() || !v2.IsValid() {
		//fmt.Println("	invalid", v1.IsValid(), v2.IsValid())
		return false
	}
	if !v2.CanSet() {
		//fmt.Println("	cannot set")
		return false
	}

	//if v1.Kind() != v2.Kind() {
	//	return false
	//}
	//
	////同じ型の別名は変換する
	//if v1.Type() != v2.Type() {
	//	v1 = v1.Convert(v2.Type())
	//}

	if !v1.Type().ConvertibleTo(v2.Type()) {
		//fmt.Printf("copy:not ConvertibleTo type! kind to:%v, from:%v type to:%v, from:%v\n", v2.Kind(), v1.Kind(), v2.Type(), v1.Type())
		return false
	}

	//if !v1.Type().AssignableTo(v2.Type()) {
	//	fmt.Printf("copy:not AssignableTo type! kind to:%v, from:%v type to:%v, from:%v\n", v2.Kind(), v1.Kind(), v2.Type(), v1.Type())
	//	return false
	//}

	v1 = v1.Convert(v2.Type())

	//XXX struct同士の場合入れ子辿って再帰でディープコピーとか
	v2.Set(v1)
	return true
}

//自動値変換とコピー
func ConvCopy(v1 reflect.Value, v2 reflect.Value) error {
	if !v1.IsValid() || !v2.IsValid() {
		return errors.New(fmt.Sprint("invalid", v1.IsValid(), v2.IsValid()))
	}
	if !v2.CanSet() {
		return errors.New(fmt.Sprint("cannot set"))
	}

	//any type to string
	if v2.Kind() == reflect.String && v1.Kind() != reflect.String {
		v1 = reflect.ValueOf(fmt.Sprint(v1.Interface()))
	}

	//string to common type
	if v1.Kind() == reflect.String && v2.Kind() != reflect.String {
		switch v2.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			v1v, err := strconv.ParseUint(fmt.Sprint(v1.Interface()), 10, 64)
			if err != nil {
				return errors.Wrap(err)
			}
			v1 = reflect.ValueOf(v1v)
		case reflect.Float32, reflect.Float64:
			v1v, err := strconv.ParseFloat(fmt.Sprint(v1.Interface()), 64)
			if err != nil {
				return errors.Wrap(err)
			}
			v1 = reflect.ValueOf(v1v)
		case reflect.Bool:
			v1v, err := strconv.ParseBool(fmt.Sprint(v1.Interface()))
			if err != nil {
				return errors.Wrap(err)
			}
			v1 = reflect.ValueOf(v1v)
		}
	}

	//型変換はConvertに任せる
	if !v1.Type().ConvertibleTo(v2.Type()) {
		return errors.New(fmt.Sprintf("copy kind to:%v, from:%v type to:%v, from:%v\n", v2.Kind(), v1.Kind(), v2.Type(), v1.Type()))
	}
	v1 = v1.Convert(v2.Type())

	v2.Set(v1)
	return nil
}

func RefToRef(x1 interface{}, x2 interface{}) bool {
	v1 := reflect.ValueOf(x1).Elem()
	v2 := reflect.ValueOf(x2).Elem()
	return copy(v1, v2)
}

func ValueToRef(v1 reflect.Value, x2 interface{}) bool {
	v2 := reflect.ValueOf(x2).Elem()
	return copy(v1, v2)
}

func RefToValue(x1 interface{}, v2 reflect.Value) bool {
	v1 := reflect.ValueOf(x1).Elem()
	return copy(v1, v2)
}

func FieldToRef(x1 interface{}, field string, x2 interface{}) bool {
	v1 := reflect.ValueOf(x1).Elem().FieldByName(field)
	v2 := reflect.ValueOf(x2).Elem()
	return copy(v1, v2)
}
func FieldToValue(x1 interface{}, field string, v2 reflect.Value) bool {
	v1 := reflect.ValueOf(x1).Elem().FieldByName(field)
	return copy(v1, v2)
}
func RefToField(x1 interface{}, x2 interface{}, field string) bool {
	v1 := reflect.ValueOf(x1).Elem()
	v2 := reflect.ValueOf(x2).Elem().FieldByName(field)
	return copy(v1, v2)
}
func ValueToField(v1 reflect.Value, x2 interface{}, field string) bool {
	v2 := reflect.ValueOf(x2).Elem().FieldByName(field)
	return copy(v1, v2)
}

//同名、同Typeフィールドの値をコピー
func StructToStruct(x1 interface{}, x2 interface{}) {
	v1 := reflect.ValueOf(x1).Elem()
	t1 := v1.Type()
	v2 := reflect.ValueOf(x2).Elem()
	t2 := v2.Type()

	for i := 0; i < t1.NumField(); i++ {
		f1 := t1.Field(i)
		if f2, ok := t2.FieldByName(f1.Name); ok {
			fv1 := v1.FieldByIndex(f1.Index)
			fv2 := v2.FieldByIndex(f2.Index)

			//r := copy(fv1, fv2)
			//fmt.Println(f1.Name, r)
			copy(fv1, fv2)
		}
	}
}

func StructToMap(x1 interface{}) map[string]interface{} {
	res := make(map[string]interface{})
	v1 := reflect.ValueOf(x1).Elem()
	t1 := v1.Type()

	for i := 0; i < t1.NumField(); i++ {
		f1 := t1.Field(i)
		f := v1.FieldByIndex(f1.Index)
		if f.Kind() == reflect.Ptr && f.IsNil() {
			continue
		}
		res[f1.Name] = f.Interface()
	}
	return res
}

func MapToStructWithTag(x1 map[string]interface{}, x2 interface{}, tagName string) {
	v2 := reflect.ValueOf(x2).Elem()
	t2 := v2.Type()

	for i := 0; i < t2.NumField(); i++ {
		f2 := t2.Field(i)
		key := f2.Tag.Get(tagName)
		if _, ok := x1[key]; !ok || key == "" || x1[key] == nil {
			continue
		}
		//fmt.Println(f2.Name, key, x1[key])
		v1 := reflect.ValueOf(x1[key])
		err := ConvCopy(v1, v2.FieldByIndex(f2.Index))
		if err != nil {
			fmt.Println(err)
		}
	}
}
