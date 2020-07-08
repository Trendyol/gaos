/*
Copyright 2020 The Gaos Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package executor

import (
	"reflect"
	"strings"
)

type Config struct {
	Scenario          string
	Environment       string `for:"all"`
	Registry          string `for:"all"`
	Username          string `for:"all"`
	Password          string `for:"all"`
	Timeout           string `for:"docker"`
	Config            string `for:"k8s"`
	Namespace         string `for:"k8s"`
	Cpu               string `for:"k8s"`
	Memory            string `for:"k8s"`
	Secret            string `for:"k8s"`
	Replica           string `for:"k8s"`
	ContinueOnFailure int    `for:"all"`
}

func (e *Config) GetMyConfig(my string) map[string]string {

	r := reflect.ValueOf(*e)
	result := map[string]string{}

	for i := 0; i < r.NumField(); i++ {
		field := r.Type().Field(i)

		if v, ok := field.Tag.Lookup("for"); ok {
			arr := strings.Split(v, ",")

			for _, v := range arr {
				if v == "all" || v == my {
					result[field.Name] = r.Field(i).String()
				}
			}

		}

	}

	return result
}
