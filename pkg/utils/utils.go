package utils

import "reflect"

func StructToMap(i interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	v := reflect.ValueOf(i)

	// Dereference pointer if necessary.
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		result[field.Name] = v.Field(i).Interface()
	}
	return result
}
