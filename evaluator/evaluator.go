package evaluator

import (
	"fmt"
	"monkey/ast"
	"monkey/object"
)

var (
	NULL  = &object.Null{}
	TRUE  = &object.Boolean{Value: true}
	FALSE = &object.Boolean{Value: false}
)

func Eval(node ast.Node, env *object.Environment) object.Object {
	switch node := node.(type) {
	// プログラム
	case *ast.Program:
		return evalProgram(node, env)

	// 式
	case *ast.ExpressionStatement:
		return Eval(node.Expression, env)

	// 整数リテラル
	case *ast.IntegerLiteral:
		return &object.Integer{Value: node.Value}

	// 真偽値
	case *ast.Boolean:
		return nativeBoolToBooleanObject(node.Value)

	// 前置式
	case *ast.PrefixExpression:
		right := Eval(node.Right, env)
		if isError(right) {
			return right
		}
		return evalPrefixExpression(node.Operator, right)

	// 中置式
	case *ast.InfixExpression:
		left := Eval(node.Left, env)
		if isError(left) {
			return left
		}

		right := Eval(node.Right, env)
		if isError(right) {
			return right
		}
		return evalInfixExpression(node.Operator, left, right)

	// ブロック文
	case *ast.BlockStatement:
		return evalBlockStatement(node, env)

	// if式
	case *ast.IfExpression:
		return evalIfExpression(node, env)

	// return文
	case *ast.ReturnStatement:
		val := Eval(node.ReturnValue, env)
		if isError(val) {
			return val
		}
		return &object.ReturnValue{Value: val}

	// let文
	case *ast.LetStatement:
		val := Eval(node.Value, env)
		if isError(val) {
			return val
		}
		env.Set(node.Name.Value, val)

	// 識別子
	case *ast.Identifier:
		return evalIdentifier(node, env)
	}

	return nil
}

/*
プログラムを評価
*/
func evalProgram(program *ast.Program, env *object.Environment) object.Object {
	var result object.Object

	for _, statement := range program.Statements {
		result = Eval(statement, env)

		switch result := result.(type) {
		case *object.ReturnValue:
			return result.Value
		case *object.Error:
			return result
		}
	}

	return result
}

/*
文を評価
*/
func evalStatements(stmts []ast.Statement, env *object.Environment) object.Object {
	var result object.Object

	for _, statement := range stmts {
		// 文を評価
		result = Eval(statement, env)

		// 文の評価結果が戻り値かどうか確認
		if returnValue, ok := result.(*object.ReturnValue); ok {
			// 戻り値の場合は以降の文の評価を打ち切り戻り値の値を返す
			return returnValue.Value
		}
	}

	return result
}

func evalPrefixExpression(operator string, right object.Object) object.Object {
	switch operator {
	case "!":
		return evalBangOperatorExpression(right)
	case "-":
		return evalMinusPrefixOperatorExpression(right)
	default:
		return newError("unknown operator: %s%s", operator, right.Type())
	}
}

func evalBangOperatorExpression(right object.Object) object.Object {
	switch right {
	case TRUE:
		return FALSE
	case FALSE:
		return TRUE
	case NULL:
		return TRUE
	default:
		return FALSE
	}
}

func evalMinusPrefixOperatorExpression(right object.Object) object.Object {
	if right.Type() != object.INTEGER_OBJ {
		return newError("unknown operator: -%s", right.Type())
	}

	value := right.(*object.Integer).Value
	return &object.Integer{Value: -value}

}

/*
中置式を評価
*/
func evalInfixExpression(
	operator string,
	left, right object.Object,
) object.Object {
	switch {
	// 左辺、右辺共に整数の場合
	case left.Type() == object.INTEGER_OBJ && right.Type() == object.INTEGER_OBJ:
		// 整数同士を評価して結果を返す
		return evalIntegerInfixExpression(operator, left, right)
	//
	case operator == "==":
		return nativeBoolToBooleanObject(left == right)
	case operator == "!=":
		return nativeBoolToBooleanObject(left != right)
	case left.Type() != right.Type():
		return newError("type mismatch: %s %s %s", left.Type(), operator, right.Type())
	default:
		return newError("unknown operator: %s %s %s", left.Type(), operator, right.Type())
	}
}

/*
整数同士の中置式を評価
*/
func evalIntegerInfixExpression(
	operator string,
	left, right object.Object,
) object.Object {
	// 左辺の値を取り出す
	leftVal := left.(*object.Integer).Value
	// 右辺の値を取り出す
	rightVal := right.(*object.Integer).Value

	// 演算子で分岐
	switch operator {
	case "+":
		return &object.Integer{Value: leftVal + rightVal}
	case "-":
		return &object.Integer{Value: leftVal - rightVal}
	case "*":
		return &object.Integer{Value: leftVal * rightVal}
	case "/":
		return &object.Integer{Value: leftVal / rightVal}
	case "<":
		return nativeBoolToBooleanObject(leftVal < rightVal)
	case ">":
		return nativeBoolToBooleanObject(leftVal > rightVal)
	case "==":
		return nativeBoolToBooleanObject(leftVal == rightVal)
	case "!=":
		return nativeBoolToBooleanObject(leftVal != rightVal)
	default:
		return newError("unknown operator: %s %s %s", left.Type(), operator, right.Type())
	}
}

func nativeBoolToBooleanObject(input bool) *object.Boolean {
	if input {
		return TRUE
	}
	return FALSE
}

func evalIfExpression(ie *ast.IfExpression, env *object.Environment) object.Object {
	condition := Eval(ie.Condition, env)
	if isError(condition) {
		return condition
	}

	if isTruthy(condition) {
		return Eval(ie.Consequence, env)
	} else if ie.Alternative != nil {
		return Eval(ie.Alternative, env)
	} else {
		return NULL
	}
}

/*
ブロック文を評価
*/
func evalBlockStatement(block *ast.BlockStatement, env *object.Environment) object.Object {
	var result object.Object

	// ノードに含まれている全ての文を評価
	for _, statement := range block.Statements {
		// 文を評価して結果を取得
		result = Eval(statement, env)

		if result != nil {
			rt := result.Type()
			if rt == object.RETURN_VALUE_OBJ || rt == object.ERROR_OBJ {
				return result
			}
		}
	}

	return result
}

/*
識別子を評価
*/
func evalIdentifier(
	node *ast.Identifier,
	env *object.Environment,
) object.Object {
	// 環境から識別子を取得
	val, ok := env.Get(node.Value)
	if !ok {
		// 識別子が見つからなかった場合はエラーオブジェクトを返す
		return newError("identifier not found: " + node.Value)
	}

	// 識別子が見つかった場合はそれを返す
	return val
}

func isTruthy(obj object.Object) bool {
	switch obj {
	case NULL:
		return false
	case TRUE:
		return true
	case FALSE:
		return false
	default:
		return true
	}
}

/*
エラー判定
*/
func isError(obj object.Object) bool {
	if obj != nil {
		return obj.Type() == object.ERROR_OBJ
	}
	return false
}

func newError(format string, a ...interface{}) *object.Error {
	return &object.Error{Message: fmt.Sprintf(format, a...)}
}
