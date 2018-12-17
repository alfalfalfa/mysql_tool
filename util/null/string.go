package null

import (
	"fmt"
	"gopkg.in/guregu/null.v3"
	"reflect"
)

type String struct {
	null.String
}

// StringFrom creates a new String that will never be blank.
func StringFrom(s string) String {
	return String{null.NewString(s, true)}
}
func NullString() String {
	return String{null.NewString("", false)}
}

// MarshalYAML implements yaml.Marshaler.
// It will encode null if this String is null.
func (s String) MarshalYAML() (interface{}, error) {
	if !s.Valid {
		return nil, nil
	}
	return s.String.String, nil
}

// UnmarshalYAML implements yaml.Unmarshaler.
// It supports string and null input. Blank string input does not produce a null String.
// It also supports unmarshalling a sql.NullString.
func (s *String) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var err error
	var v interface{}
	if err = unmarshal(&v); err != nil {
		return err
	}
	switch x := v.(type) {
	case string:
		s.String.String = x
	case map[string]interface{}:
		err = unmarshal(&s.NullString)
	case nil:
		s.Valid = false
		return nil
	default:
		err = fmt.Errorf("yaml: cannot unmarshal %v into Go value of type null.String", reflect.TypeOf(v).Name())
	}
	s.Valid = err == nil
	return err
}
