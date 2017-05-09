package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
)

type visitor struct {
	constSpecs []string
	funcDecls  []string
	typeSpecs  []string
	warnings   []string
}

func (v *visitor) Visit(node ast.Node) ast.Visitor {
	switch typedNode := node.(type) {
	case *ast.File:
		return v
	case *ast.GenDecl:
		if typedNode.Tok == token.CONST {
			constName := typedNode.Specs[0].(*ast.ValueSpec).Names[0].Name
			v.constSpecs = append(v.constSpecs, constName)
			if len(v.funcDecls) != 0 {
				v.warnings = append(v.warnings, fmt.Sprintf("constant '%s' comes after a function declaration", constName))
			}
		}
		return v
	case *ast.TypeSpec:
		v.typeSpecs = append(v.typeSpecs, typedNode.Name.Name)
		if len(v.funcDecls) != 0 {
			v.warnings = append(v.warnings, fmt.Sprintf("type declaration for '%s' comes after a function declaration", typedNode.Name.Name))
		}
	case *ast.FuncDecl:
		v.funcDecls = append(v.funcDecls, typedNode.Name.Name)
	}

	return nil
}

func main() {
	src := `
package foo


func f1() bool {
	type foo struct {
	}
	return true
}

type type1 struct {
}

const a = 6
`

	fileSet := token.NewFileSet()
	f, err := parser.ParseFile(fileSet, "", src, 0)
	if err != nil {
		panic(err)
	}

	v := visitor{}

	ast.Walk(&v, f)

	// fmt.Println(ast.Print(fileSet, f))

	for _, warning := range v.warnings {
		fmt.Println(warning)
	}
}
