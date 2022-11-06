package reflector

import "reflect"

func ConvertType(value interface{}, dstType reflect.Kind) interface{} {
	switch dstType {
	case reflect.Uint8:
		return uint8(value.(float64))
	case reflect.Uint16:
		return uint16(value.(float64))
	case reflect.Uint32:
		return uint32(value.(float64))
	case reflect.Uint64:
		return uint64(value.(float64))
	case reflect.Int8:
		return int8(value.(float64))
	case reflect.Int16:
		return int16(value.(float64))
	case reflect.Int32:
		return int32(value.(float64))
	case reflect.Int64:
		return int64(value.(float64))
	}

	return nil
}
