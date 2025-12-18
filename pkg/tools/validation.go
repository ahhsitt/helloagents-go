package tools

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
)

// Validate 验证参数是否符合 Schema
func Validate(schema ParameterSchema, args map[string]interface{}) error {
	// 验证必需参数
	for _, req := range schema.Required {
		if _, ok := args[req]; !ok {
			return fmt.Errorf("missing required parameter: %s", req)
		}
	}

	// 验证每个参数的类型和约束
	for name, value := range args {
		propSchema, exists := schema.Properties[name]
		if !exists {
			if !schema.AdditionalProperties {
				return fmt.Errorf("unexpected parameter: %s", name)
			}
			continue
		}

		if err := validateProperty(name, propSchema, value); err != nil {
			return err
		}
	}

	return nil
}

// validateProperty 验证单个属性
func validateProperty(name string, schema PropertySchema, value interface{}) error {
	if value == nil {
		return nil // nil 值跳过类型检查
	}

	// 类型检查
	if err := validateType(name, schema.Type, value); err != nil {
		return err
	}

	// 约束检查
	switch schema.Type {
	case "string":
		return validateString(name, schema, value.(string))
	case "number", "integer":
		return validateNumber(name, schema, value)
	case "array":
		return validateArray(name, schema, value)
	}

	return nil
}

// validateType 验证值类型
func validateType(name, expectedType string, value interface{}) error {
	v := reflect.ValueOf(value)
	kind := v.Kind()

	switch expectedType {
	case "string":
		if kind != reflect.String {
			return fmt.Errorf("parameter %s: expected string, got %T", name, value)
		}
	case "number":
		if kind != reflect.Float64 && kind != reflect.Float32 &&
			kind != reflect.Int && kind != reflect.Int64 && kind != reflect.Int32 {
			return fmt.Errorf("parameter %s: expected number, got %T", name, value)
		}
	case "integer":
		if kind != reflect.Int && kind != reflect.Int64 && kind != reflect.Int32 {
			// JSON 数字可能解析为 float64
			if kind == reflect.Float64 {
				f := value.(float64)
				if f != float64(int64(f)) {
					return fmt.Errorf("parameter %s: expected integer, got float", name)
				}
			} else {
				return fmt.Errorf("parameter %s: expected integer, got %T", name, value)
			}
		}
	case "boolean":
		if kind != reflect.Bool {
			return fmt.Errorf("parameter %s: expected boolean, got %T", name, value)
		}
	case "array":
		if kind != reflect.Slice && kind != reflect.Array {
			return fmt.Errorf("parameter %s: expected array, got %T", name, value)
		}
	case "object":
		if kind != reflect.Map {
			return fmt.Errorf("parameter %s: expected object, got %T", name, value)
		}
	}

	return nil
}

// validateString 验证字符串约束
func validateString(name string, schema PropertySchema, value string) error {
	// 长度验证
	if schema.MinLength != nil && len(value) < *schema.MinLength {
		return fmt.Errorf("parameter %s: length %d is less than minimum %d",
			name, len(value), *schema.MinLength)
	}
	if schema.MaxLength != nil && len(value) > *schema.MaxLength {
		return fmt.Errorf("parameter %s: length %d exceeds maximum %d",
			name, len(value), *schema.MaxLength)
	}

	// 模式验证
	if schema.Pattern != "" {
		matched, err := regexp.MatchString(schema.Pattern, value)
		if err != nil {
			return fmt.Errorf("parameter %s: invalid pattern %s: %v",
				name, schema.Pattern, err)
		}
		if !matched {
			return fmt.Errorf("parameter %s: value does not match pattern %s",
				name, schema.Pattern)
		}
	}

	// 枚举验证
	if len(schema.Enum) > 0 {
		found := false
		for _, e := range schema.Enum {
			if e == value {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("parameter %s: value '%s' not in allowed values %v",
				name, value, schema.Enum)
		}
	}

	return nil
}

// validateNumber 验证数值约束
func validateNumber(name string, schema PropertySchema, value interface{}) error {
	var num float64

	switch v := value.(type) {
	case float64:
		num = v
	case float32:
		num = float64(v)
	case int:
		num = float64(v)
	case int64:
		num = float64(v)
	case int32:
		num = float64(v)
	default:
		return fmt.Errorf("parameter %s: cannot convert %T to number", name, value)
	}

	if schema.Minimum != nil && num < *schema.Minimum {
		return fmt.Errorf("parameter %s: value %g is less than minimum %g",
			name, num, *schema.Minimum)
	}
	if schema.Maximum != nil && num > *schema.Maximum {
		return fmt.Errorf("parameter %s: value %g exceeds maximum %g",
			name, num, *schema.Maximum)
	}

	return nil
}

// validateArray 验证数组约束
func validateArray(name string, schema PropertySchema, value interface{}) error {
	v := reflect.ValueOf(value)
	length := v.Len()

	// 如果有元素 Schema，验证每个元素
	if schema.Items != nil {
		for i := 0; i < length; i++ {
			elem := v.Index(i).Interface()
			elemName := name + "[" + strconv.Itoa(i) + "]"
			if err := validateProperty(elemName, *schema.Items, elem); err != nil {
				return err
			}
		}
	}

	return nil
}
