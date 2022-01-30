package utils

import (
	"math"
	"reflect"
)

func setInt(f *reflect.Value, value interface{}) {
	switch cmd := value.(type) {
	case int:
		f.SetInt(int64(cmd))
	case int8:
		f.SetInt(int64(cmd))
	case int16:
		f.SetInt(int64(cmd))
	case int32:
		f.SetInt(int64(cmd))
	case uint:
		f.SetInt(int64(cmd))
	case uint8:
		f.SetInt(int64(cmd))
	case uint16:
		f.SetInt(int64(cmd))
	case uint32:
		f.SetInt(int64(cmd))
	case uint64:
		f.SetInt(int64(cmd))
	case float32:
		f.SetInt(int64(cmd))
	case float64:
		f.SetInt(int64(cmd))
	}
}

func setUint(f *reflect.Value, value interface{}) {
	switch cmd := value.(type) {
	case int:
		f.SetUint(uint64(math.Abs(float64(cmd))))
	case int8:
		f.SetUint(uint64(math.Abs(float64(cmd))))
	case int16:
		f.SetUint(uint64(math.Abs(float64(cmd))))
	case int32:
		f.SetUint(uint64(math.Abs(float64(cmd))))
	case int64:
		f.SetUint(uint64(math.Abs(float64(cmd))))
	case uint:
		f.SetUint(uint64(math.Abs(float64(cmd))))
	case uint8:
		f.SetUint(uint64(math.Abs(float64(cmd))))
	case uint16:
		f.SetUint(uint64(math.Abs(float64(cmd))))
	case uint32:
		f.SetUint(uint64(math.Abs(float64(cmd))))
	case uint64:
		f.SetUint(uint64(math.Abs(float64(cmd))))
	case float32:
		f.SetUint(uint64(math.Abs(float64(cmd))))
	case float64:
		f.SetUint(uint64(math.Abs(float64(cmd))))
	}
}

func setStructPropertyByNamne(name string, value interface{}, dst interface{}) {
	dstValue := reflect.ValueOf(dst)
	s := dstValue.Elem()
	if s.Kind() == reflect.Struct {
		f := s.FieldByName(name)
		if f.IsValid() {
			if f.CanSet() {
				switch f.Kind() {
				case reflect.Uint:
					setUint(&f, value)
				case reflect.Uint8:
					setUint(&f, value)
				case reflect.Uint16:
					setUint(&f, value)
				case reflect.Uint32:
					setUint(&f, value)
				case reflect.Uint64:
					setUint(&f, value)
				case reflect.Int:
					setInt(&f, value)
				case reflect.Int8:
					setInt(&f, value)
				case reflect.Int16:
					setInt(&f, value)
				case reflect.Int32:
					setInt(&f, value)
				case reflect.Int64:
					setInt(&f, value)
				}
			}
		}
	}
}

func SetStructProperties(srcMap map[string]interface{}, dst interface{}) {
	for key, value := range srcMap {
		setStructPropertyByNamne(key, value, dst)
	}
}
