package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
)

type visitor struct {
	constSpecs []string
	funcDecls  []string
	typeSpecs  []string
	varSpecs   []string
	warnings   []string
}

func (v *visitor) Visit(node ast.Node) ast.Visitor {
	switch typedNode := node.(type) {
	case *ast.File:
		return v
	case *ast.GenDecl:
		if typedNode.Tok == token.CONST {
			v.checkConst(typedNode)
		} else if typedNode.Tok == token.VAR {
			v.checkVar(typedNode)
		}
		return v
	case *ast.FuncDecl:
		v.checkFunc(typedNode)
	case *ast.TypeSpec:
		v.checkType(typedNode)
	}

	return nil
}

func (v *visitor) checkConst(node *ast.GenDecl) {
	constName := node.Specs[0].(*ast.ValueSpec).Names[0].Name
	v.constSpecs = append(v.constSpecs, constName)
	if len(v.funcDecls) != 0 {
		v.warnings = append(v.warnings, fmt.Sprintf("constant '%s' comes after a function declaration", constName))
	}
	if len(v.typeSpecs) != 0 {
		v.warnings = append(v.warnings, fmt.Sprintf("constant '%s' comes after a type declaration", constName))
	}
	if len(v.varSpecs) != 0 {
		v.warnings = append(v.warnings, fmt.Sprintf("constant '%s' comes after a variable declaration", constName))
	}
}

func (v *visitor) checkVar(node *ast.GenDecl) {
	varName := node.Specs[0].(*ast.ValueSpec).Names[0].Name
	v.varSpecs = append(v.varSpecs, varName)
	if len(v.funcDecls) != 0 {
		v.warnings = append(v.warnings, fmt.Sprintf("variable '%s' comes after a function declaration", varName))
	}
	if len(v.typeSpecs) != 0 {
		v.warnings = append(v.warnings, fmt.Sprintf("variable '%s' comes after a type declaration", varName))
	}
}

func (v *visitor) checkFunc(node *ast.FuncDecl) {
	funcName := node.Name.Name
	v.funcDecls = append(v.funcDecls, funcName)
}

func (v *visitor) checkType(node *ast.TypeSpec) {
	typeName := node.Name.Name
	v.typeSpecs = append(v.typeSpecs, typeName)
	if len(v.funcDecls) != 0 {
		v.warnings = append(v.warnings, fmt.Sprintf("type declaration for '%s' comes after a function declaration", typeName))
	}
}

func main() {
	fmt.Printf("checking %s\n", os.Args[1])
	fileSet := token.NewFileSet()
	f, err := parser.ParseFile(fileSet, os.Args[1], nil, 0)
	if err != nil {
		panic(err)
	}

	v := visitor{}

	ast.Walk(&v, f)

	// fmt.Println(ast.Print(fileSet, f))

	for _, warning := range v.warnings {
		fmt.Printf("  %s\n", warning)
	}

	if len(v.warnings) > 0 {
		os.Exit(1)
	}
}
