package binding

import (
	"errors"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
)

type QueryBinding struct {
	Tag string
}

func (q QueryBinding) Bind(r *http.Request, obj any) error {
	values := r.URL.Query()
	return reflectMapToObj(values, q.Tag, obj)
}

func (q QueryBinding) Name() string {
	return "query"
}

// reflectMapToObj 将 URL 查询参数映射到结构体
func reflectMapToObj(values url.Values, tag string, obj any) error {
	if len(values) == 0 {
		return nil
	}

	if tag == "" {
		return errors.New("bind query failed: empty tag provided")
	}

	rt := reflect.TypeOf(obj)
	rv := reflect.ValueOf(obj)

	if rt.Kind() != reflect.Ptr {
		return errors.New("bind query failed: obj must be a pointer")
	}

	if rt.Elem().Kind() != reflect.Struct {
		return errors.New("bind query failed: obj must be a pointer of struct")
	}

	// 取消指针，获取结构体的类型和值
	elemType := rt.Elem()
	elemValue := rv.Elem()

	// 遍历结构体字段
	for i := 0; i < elemType.NumField(); i++ {
		field := elemType.Field(i)
		fieldName := field.Name
		fieldValue := elemValue.Field(i)

		// 获取结构体标签的查询参数名
		queryKey := field.Tag.Get(tag)
		if queryKey == "-" {
			continue
		}

		if queryKey == "" {
			queryKey = fieldName // 如果标签为空，使用字段名作为查询参数名
		}

		// 跳过不可导出的字段
		if !fieldValue.CanSet() {
			continue
		}

		// 检查查询参数是否存在
		paramValues := values[queryKey]
		if len(paramValues) == 0 {
			continue
		}

		paramValue := paramValues[0] // 取第一个值

		// 不处理嵌套结构体
		if fieldValue.Kind() == reflect.Struct {
			continue
		}

		// 处理基本类型
		switch fieldValue.Kind() {
		case reflect.String:
			fieldValue.SetString(paramValue)
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if v, err := strconv.ParseInt(paramValue, 10, 64); err == nil {
				fieldValue.SetInt(v)
			} else {
				return errors.New("bind query failed: invalid int value for field " + fieldName)
			}
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			if v, err := strconv.ParseUint(paramValue, 10, 64); err == nil {
				fieldValue.SetUint(v)
			} else {
				return errors.New("bind query failed: invalid uint value for field " + fieldName)
			}
		case reflect.Bool:
			if v, err := strconv.ParseBool(paramValue); err == nil {
				fieldValue.SetBool(v)
			} else {
				return errors.New("bind query failed: invalid bool value for field " + fieldName)
			}
		case reflect.Float32, reflect.Float64:
			if v, err := strconv.ParseFloat(paramValue, 64); err == nil {
				fieldValue.SetFloat(v)
			} else {
				return errors.New("bind query failed: invalid float value for field " + fieldName)
			}
		case reflect.Slice:
			// 如果字段是 Slice 或 Array，需要解析多个值
			if len(paramValues) > 0 {
				// 仅处理字符串类型的 Slice
				if fieldType := field.Type.Elem(); fieldType.Kind() == reflect.String {
					slice := reflect.MakeSlice(reflect.SliceOf(reflect.TypeOf("")), 0, len(paramValues))
					for _, v := range paramValues {
						slice = reflect.Append(slice, reflect.ValueOf(v))
					}
					fieldValue.Set(slice)
				} else {
					return errors.New("bind query failed: unsupported slice/array type for field " + fieldName)
				}
			}
		default:
			return errors.New("bind query failed: unsupported field type " + fieldValue.Kind().String() + " for field " + fieldName)
		}
	}

	return nil
}
