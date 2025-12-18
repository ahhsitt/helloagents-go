package tools

import (
	"fmt"
	"reflect"
	"strings"
)

// GenerateDescription 自动生成工具描述
//
// 基于工具名称和参数 Schema 生成描述文本。
func GenerateDescription(name string, schema ParameterSchema) string {
	var sb strings.Builder

	// 基础描述
	sb.WriteString(fmt.Sprintf("Tool: %s\n", name))

	if len(schema.Properties) == 0 {
		sb.WriteString("No parameters required.\n")
		return sb.String()
	}

	sb.WriteString("Parameters:\n")

	// 参数描述
	for propName, prop := range schema.Properties {
		required := ""
		for _, req := range schema.Required {
			if req == propName {
				required = " (required)"
				break
			}
		}

		sb.WriteString(fmt.Sprintf("  - %s (%s)%s", propName, prop.Type, required))

		if prop.Description != "" {
			sb.WriteString(": " + prop.Description)
		}

		if len(prop.Enum) > 0 {
			sb.WriteString(fmt.Sprintf(" [allowed: %s]", strings.Join(prop.Enum, ", ")))
		}

		sb.WriteString("\n")
	}

	return sb.String()
}

// SchemaFromStruct 从结构体类型生成参数 Schema
//
// 使用示例:
//
//	type WeatherParams struct {
//	    Location string `json:"location" desc:"City name"`
//	    Unit     string `json:"unit" desc:"Temperature unit (celsius/fahrenheit)"`
//	}
//	schema := tools.SchemaFromStruct(WeatherParams{})
func SchemaFromStruct(v interface{}) ParameterSchema {
	t := reflect.TypeOf(v)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	if t.Kind() != reflect.Struct {
		return ParameterSchema{Type: "object"}
	}

	props := make(map[string]PropertySchema)
	var required []string

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		// 跳过未导出的字段
		if !field.IsExported() {
			continue
		}

		// 获取 JSON 标签
		jsonTag := field.Tag.Get("json")
		if jsonTag == "" || jsonTag == "-" {
			continue
		}

		// 解析 JSON 标签
		parts := strings.Split(jsonTag, ",")
		name := parts[0]

		// 获取描述标签
		desc := field.Tag.Get("desc")

		// 获取必需标签
		requiredTag := field.Tag.Get("required")

		// 转换类型
		propSchema := fieldToSchema(field.Type)
		propSchema.Description = desc

		props[name] = propSchema

		if requiredTag == "true" || requiredTag == "1" {
			required = append(required, name)
		}
	}

	return ParameterSchema{
		Type:       "object",
		Properties: props,
		Required:   required,
	}
}

// fieldToSchema 将 Go 类型转换为 PropertySchema
func fieldToSchema(t reflect.Type) PropertySchema {
	switch t.Kind() {
	case reflect.String:
		return PropertySchema{Type: "string"}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return PropertySchema{Type: "integer"}
	case reflect.Float32, reflect.Float64:
		return PropertySchema{Type: "number"}
	case reflect.Bool:
		return PropertySchema{Type: "boolean"}
	case reflect.Slice, reflect.Array:
		elemSchema := fieldToSchema(t.Elem())
		return PropertySchema{
			Type:  "array",
			Items: &elemSchema,
		}
	case reflect.Map:
		return PropertySchema{Type: "object"}
	case reflect.Struct:
		// 递归处理嵌套结构体
		nested := SchemaFromStruct(reflect.New(t).Elem().Interface())
		return PropertySchema{
			Type:       "object",
			Properties: nested.Properties,
			Required:   nested.Required,
		}
	case reflect.Ptr:
		return fieldToSchema(t.Elem())
	default:
		return PropertySchema{Type: "string"}
	}
}

// DescribeTools 生成工具列表的描述
func DescribeTools(tools []Tool) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Available Tools (%d):\n\n", len(tools)))

	for i, tool := range tools {
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, tool.Name()))
		sb.WriteString(fmt.Sprintf("   Description: %s\n", tool.Description()))

		schema := tool.Parameters()
		if len(schema.Properties) > 0 {
			sb.WriteString("   Parameters:\n")
			for propName, prop := range schema.Properties {
				required := ""
				for _, req := range schema.Required {
					if req == propName {
						required = "*"
						break
					}
				}
				sb.WriteString(fmt.Sprintf("     - %s%s (%s): %s\n",
					propName, required, prop.Type, prop.Description))
			}
		}
		sb.WriteString("\n")
	}

	return sb.String()
}
