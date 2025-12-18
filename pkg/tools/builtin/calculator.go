// Package builtin 提供框架内置的常用工具
package builtin

import (
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"

	"github.com/easyops/helloagents-go/pkg/tools"
)

// Calculator 计算器工具
//
// 支持基础四则运算和数学表达式计算。
type Calculator struct{}

// NewCalculator 创建计算器工具
func NewCalculator() *Calculator {
	return &Calculator{}
}

// Name 返回工具名称
func (c *Calculator) Name() string {
	return "calculator"
}

// Description 返回工具描述
func (c *Calculator) Description() string {
	return "Perform mathematical calculations. Supports basic arithmetic operations (+, -, *, /) and parentheses."
}

// Parameters 返回参数 Schema
func (c *Calculator) Parameters() tools.ParameterSchema {
	return tools.ParameterSchema{
		Type: "object",
		Properties: map[string]tools.PropertySchema{
			"expression": {
				Type:        "string",
				Description: "The mathematical expression to evaluate, e.g., '2 + 3 * 4' or '(10 - 5) / 2'",
			},
		},
		Required: []string{"expression"},
	}
}

// Execute 执行计算
func (c *Calculator) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	exprRaw, ok := args["expression"]
	if !ok {
		return "", fmt.Errorf("missing required parameter: expression")
	}

	expr, ok := exprRaw.(string)
	if !ok {
		return "", fmt.Errorf("expression must be a string")
	}

	result, err := evalExpression(expr)
	if err != nil {
		return "", fmt.Errorf("failed to evaluate expression: %w", err)
	}

	return fmt.Sprintf("%g", result), nil
}

// evalExpression 使用 Go AST 安全地计算数学表达式
func evalExpression(expr string) (float64, error) {
	// 解析表达式
	node, err := parser.ParseExpr(expr)
	if err != nil {
		return 0, fmt.Errorf("invalid expression: %w", err)
	}

	return evalNode(node)
}

func evalNode(node ast.Expr) (float64, error) {
	switch n := node.(type) {
	case *ast.BasicLit:
		// 数字字面量
		if n.Kind == token.INT || n.Kind == token.FLOAT {
			return strconv.ParseFloat(n.Value, 64)
		}
		return 0, fmt.Errorf("unsupported literal type: %v", n.Kind)

	case *ast.BinaryExpr:
		// 二元表达式
		left, err := evalNode(n.X)
		if err != nil {
			return 0, err
		}
		right, err := evalNode(n.Y)
		if err != nil {
			return 0, err
		}

		switch n.Op {
		case token.ADD:
			return left + right, nil
		case token.SUB:
			return left - right, nil
		case token.MUL:
			return left * right, nil
		case token.QUO:
			if right == 0 {
				return 0, fmt.Errorf("division by zero")
			}
			return left / right, nil
		case token.REM:
			if right == 0 {
				return 0, fmt.Errorf("modulo by zero")
			}
			return float64(int64(left) % int64(right)), nil
		default:
			return 0, fmt.Errorf("unsupported operator: %v", n.Op)
		}

	case *ast.ParenExpr:
		// 括号表达式
		return evalNode(n.X)

	case *ast.UnaryExpr:
		// 一元表达式
		x, err := evalNode(n.X)
		if err != nil {
			return 0, err
		}

		switch n.Op {
		case token.SUB:
			return -x, nil
		case token.ADD:
			return x, nil
		default:
			return 0, fmt.Errorf("unsupported unary operator: %v", n.Op)
		}

	default:
		return 0, fmt.Errorf("unsupported expression type: %T", node)
	}
}

// Validate 验证参数
func (c *Calculator) Validate(args map[string]interface{}) error {
	expr, ok := args["expression"]
	if !ok {
		return fmt.Errorf("missing required parameter: expression")
	}
	if _, ok := expr.(string); !ok {
		return fmt.Errorf("expression must be a string")
	}
	return nil
}

// compile-time interface check
var _ tools.Tool = (*Calculator)(nil)
var _ tools.ToolWithValidation = (*Calculator)(nil)
