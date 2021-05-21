package models

import (
	"github.com/alfalfalfa/mysql_tool/util/errors"
	"gopkg.in/yaml.v2"
	"log"
)

func ToYaml(el interface{}) string {
	b, err := yaml.Marshal(el)
	if err != nil {
		log.Fatal(errors.Wrap(err))
	}
	//
	//b, err = prettyprint(b)
	//if err != nil {
	//	log.Fatal(errors.Wrap(err))
	//}
	return string(b)
}
