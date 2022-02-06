package utils

import (
	"math"
	"reflect"
)

func setInt(f *reflect.Value, value interface{}) {
	switch valueTyped := value.(type) {
	case int:
		f.SetInt(int64(valueTyped))
	case int8:
		f.SetInt(int64(valueTyped))
	case int16:
		f.SetInt(int64(valueTyped))
	case int32:
		f.SetInt(int64(valueTyped))
	case uint:
		f.SetInt(int64(valueTyped))
	case uint8:
		f.SetInt(int64(valueTyped))
	case uint16:
		f.SetInt(int64(valueTyped))
	case uint32:
		f.SetInt(int64(valueTyped))
	case uint64:
		f.SetInt(int64(valueTyped))
	case float32:
		f.SetInt(int64(valueTyped))
	case float64:
		f.SetInt(int64(valueTyped))
	}
}

func setUint(f *reflect.Value, value interface{}) {
	switch valueTyped := value.(type) {
	case int:
		f.SetUint(uint64(math.Abs(float64(valueTyped))))
	case int8:
		f.SetUint(uint64(math.Abs(float64(valueTyped))))
	case int16:
		f.SetUint(uint64(math.Abs(float64(valueTyped))))
	case int32:
		f.SetUint(uint64(math.Abs(float64(valueTyped))))
	case int64:
		f.SetUint(uint64(math.Abs(float64(valueTyped))))
	case uint:
		f.SetUint(uint64(math.Abs(float64(valueTyped))))
	case uint8:
		f.SetUint(uint64(math.Abs(float64(valueTyped))))
	case uint16:
		f.SetUint(uint64(math.Abs(float64(valueTyped))))
	case uint32:
		f.SetUint(uint64(math.Abs(float64(valueTyped))))
	case uint64:
		f.SetUint(uint64(math.Abs(float64(valueTyped))))
	case float32:
		f.SetUint(uint64(math.Abs(float64(valueTyped))))
	case float64:
		f.SetUint(uint64(math.Abs(float64(valueTyped))))
	}
}

func setFloat(f *reflect.Value, value interface{}) {
	switch valueTyped := value.(type) {
	case int:
		f.SetFloat(float64(valueTyped))
	case int8:
		f.SetFloat(float64(valueTyped))
	case int16:
		f.SetFloat(float64(valueTyped))
	case int32:
		f.SetFloat(float64(valueTyped))
	case int64:
		f.SetFloat(float64(valueTyped))
	case uint:
		f.SetFloat(float64(valueTyped))
	case uint8:
		f.SetFloat(float64(valueTyped))
	case uint16:
		f.SetFloat(float64(valueTyped))
	case uint32:
		f.SetFloat(float64(valueTyped))
	case uint64:
		f.SetFloat(float64(valueTyped))
	case float32:
		f.SetFloat(float64(valueTyped))
	case float64:
		f.SetFloat(float64(valueTyped))
	}
}

func setBool(f *reflect.Value, value interface{}) {
	switch valueTyped := value.(type) {
	case bool:
		f.SetBool(valueTyped)
	}
}

func setString(f *reflect.Value, value interface{}) {
	switch valueTyped := value.(type) {
	case string:
		f.SetString(valueTyped)
	}
}

func setStructPropertyByNamne(name string, value interface{}, dst interface{}) {
	dstValue := reflect.ValueOf(dst)
	s := dstValue.Elem()
	if s.Kind() == reflect.Struct {
		f := s.FieldByName(name)
		if f.IsValid() && f.CanSet() {
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
			case reflect.Float32:
				setFloat(&f, value)
			case reflect.Float64:
				setFloat(&f, value)
			case reflect.Bool:
				setBool(&f, value)
			case reflect.String:
				setString(&f, value)
			}
		}
	}
}

func SetStructProperties(srcMap map[string]interface{}, dst interface{}) {
	for key, value := range srcMap {
		setStructPropertyByNamne(key, value, dst)
	}
}
