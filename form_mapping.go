// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package binding

import (
	"encoding/json"
	"errors"
	"reflect"
	"strconv"
	"strings"
	"time"
)

func mapForm(ptr interface{}, form map[string][]string) error {
	typ := reflect.TypeOf(ptr).Elem()
	val := reflect.ValueOf(ptr).Elem()
	for i := 0; i < typ.NumField(); i++ {
		typeField := typ.Field(i)
		structField := val.Field(i)
		if !structField.CanSet() {
			continue
		}

		// get default value
		inputFieldDefault := typeField.Tag.Get("default")

		structFieldKind := structField.Kind()
		inputFieldName := typeField.Tag.Get("json")
		if inputFieldName == "" {
			inputFieldName = typeField.Tag.Get("form")
		}
		if inputFieldName == "" {
			inputFieldName = typeField.Name

			// if "form" tag is nil, we inspect if the field is a struct.
			// this would not make sense for JSON parsing but it does for a form
			// since data is flatten
			if structFieldKind == reflect.Struct {
				err := mapForm(structField.Addr().Interface(), form)
				if err != nil {
					return err
				}
				continue
			}
		}
		// omit field
		if strings.HasPrefix(inputFieldName, "-") {
			continue
		}
		if idx := strings.Index(inputFieldName, ","); idx != -1 {
			inputFieldName = inputFieldName[:idx]
		}
		inputValue, exists := form[inputFieldName]
		if !exists {
			if inputFieldDefault == "" {
				continue
			}
			if err := setWithProperType(typeField.Type, inputFieldDefault, structField); err != nil {
				return err
			}
			continue
		}

		// handle ptr field of struct
		if structFieldKind == reflect.Ptr {
			if structField.IsNil() {
				structField.Set(reflect.New(typeField.Type.Elem()))
			}
			structField = structField.Elem()
			structFieldKind = structField.Kind()
			typeField.Type = typeField.Type.Elem()
		}

		if _, isTime := structField.Interface().(time.Time); isTime {
			if err := setTimeField(inputValue[0], typeField, structField); err != nil {
				return err
			}
			continue
		}

		if err := setWithProperType(typeField.Type, inputValue[0], structField); err != nil {
			return err
		}
	}
	return nil
}

func setWithProperType(valueType reflect.Type, val string, structField reflect.Value) error {
	switch valueType.Kind() {
	case reflect.Int:
		return setIntField(val, 0, structField)
	case reflect.Int8:
		return setIntField(val, 8, structField)
	case reflect.Int16:
		return setIntField(val, 16, structField)
	case reflect.Int32:
		return setIntField(val, 32, structField)
	case reflect.Int64:
		return setIntField(val, 64, structField)
	case reflect.Uint:
		return setUintField(val, 0, structField)
	case reflect.Uint8:
		return setUintField(val, 8, structField)
	case reflect.Uint16:
		return setUintField(val, 16, structField)
	case reflect.Uint32:
		return setUintField(val, 32, structField)
	case reflect.Uint64:
		return setUintField(val, 64, structField)
	case reflect.Bool:
		return setBoolField(val, structField)
	case reflect.Float32:
		return setFloatField(val, 32, structField)
	case reflect.Float64:
		return setFloatField(val, 64, structField)
	case reflect.String:
		structField.SetString(val)
	case reflect.Struct, reflect.Map, reflect.Slice, reflect.Array:
		return setJSONField(val, valueType, structField)
	default:
		return errors.New("Unknown type")
	}
	return nil
}

func setIntField(val string, bitSize int, field reflect.Value) error {
	if val == "" {
		val = "0"
	}
	intVal, err := strconv.ParseInt(val, 10, bitSize)
	if err == nil {
		field.SetInt(intVal)
	}
	return err
}

func setUintField(val string, bitSize int, field reflect.Value) error {
	if val == "" {
		val = "0"
	}
	uintVal, err := strconv.ParseUint(val, 10, bitSize)
	if err == nil {
		field.SetUint(uintVal)
	}
	return err
}

func setBoolField(val string, field reflect.Value) error {
	if val == "" {
		val = "false"
	}
	boolVal, err := strconv.ParseBool(val)
	if err == nil {
		field.SetBool(boolVal)
	}
	return nil
}

func setFloatField(val string, bitSize int, field reflect.Value) error {
	if val == "" {
		val = "0.0"
	}
	floatVal, err := strconv.ParseFloat(val, bitSize)
	if err == nil {
		field.SetFloat(floatVal)
	}
	return err
}

func setTimeField(val string, structField reflect.StructField, value reflect.Value) error {
	timeFormat := structField.Tag.Get("time_format")
	if timeFormat == "" {
		return errors.New("Blank time format")
	}

	if val == "" {
		value.Set(reflect.ValueOf(time.Time{}))
		return nil
	}

	l := time.Local
	if isUTC, _ := strconv.ParseBool(structField.Tag.Get("time_utc")); isUTC {
		l = time.UTC
	}

	if locTag := structField.Tag.Get("time_location"); locTag != "" {
		loc, err := time.LoadLocation(locTag)
		if err != nil {
			return err
		}
		l = loc
	}

	t, err := time.ParseInLocation(timeFormat, val, l)
	if err != nil {
		return err
	}

	value.Set(reflect.ValueOf(t))
	return nil
}

// support nested struct/map/slice for GET method, as well as for Content-Type of
// application/x-www-form-urlencoded, multipart/form-data
func setJSONField(val string, valueType reflect.Type, field reflect.Value) error {
	temp := reflect.New(valueType).Interface()
	err := json.Unmarshal([]byte(val), &temp)
	if err != nil {
		return err
	}
	field.Set(reflect.ValueOf(temp).Elem())
	return nil
}
