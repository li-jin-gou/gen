package parser

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
)

// InterfaceSet ...
type InterfaceSet struct {
	Interfaces []InterfaceInfo
}

// InterfaceInfo ...
type InterfaceInfo struct {
	Name    string
	Doc     string
	Methods []*Method
}

// Method interface's method
type Method struct {
	MethodName string
	Doc        string
	Params     []Param
	Result     []Param
}

// ParseFile get interface's info from source file
func (i *InterfaceSet) ParseFile(paths []*InterfacePath) error {
	for _, path := range paths {
		for _, file := range path.Files {
			absFilePath, err := filepath.Abs(file)
			if err != nil {
				return fmt.Errorf("file not found：%s", file)
			}

			err = i.getInterfaceFromFile(absFilePath, path.Name)
			if err != nil {
				return fmt.Errorf("can't get interface from %s:%s", path.FullName, err)
			}
		}
	}
	return nil
}

// Visit ast visit function
func (i *InterfaceSet) Visit(n ast.Node) (w ast.Visitor) {
	switch n := n.(type) {
	case *ast.TypeSpec:
		if data, ok := n.Type.(*ast.InterfaceType); ok {
			r := InterfaceInfo{
				Methods: []*Method{},
			}
			methods := data.Methods.List
			r.Name = n.Name.Name
			r.Doc = n.Doc.Text()
			for _, m := range methods {
				for _, name := range m.Names {
					r.Methods = append(r.Methods, &Method{
						MethodName: name.Name,
						Doc:        m.Doc.Text(),
						Params:     getParamList(m.Type.(*ast.FuncType).Params),
						Result:     getParamList(m.Type.(*ast.FuncType).Results),
					})
				}

			}
			i.Interfaces = append(i.Interfaces, r)
		}
	}
	return i
}

// getInterfaceFromFile get interfaces
// get all interfaces from file and compare with specified name
func (i *InterfaceSet) getInterfaceFromFile(filename string, name string) error {
	fileset := token.NewFileSet()
	f, err := parser.ParseFile(fileset, filename, nil, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("can't parse file %q: %s", filename, err)
	}

	astResult := new(InterfaceSet)
	ast.Walk(astResult, f)

	for _, info := range astResult.Interfaces {
		if name == info.Name {
			i.Interfaces = append(i.Interfaces, info)
		}
	}

	return nil
}

// Param parameters in method
type Param struct { // (user model.User)
	Package   string // package's name: model
	Name      string // param's name: user
	Type      string // param's type: User
	IsArray   bool   // is array or not
	IsPointer bool   // is pointer or not
}

func (p *Param) Eq(q Param) bool {
	return p.Package == q.Package && p.Type == q.Type
}

func (p *Param) IsError() bool {
	return p.Type == "error"
}

func (p *Param) IsGenT() bool {
	return p.Package == "gen" && p.Type == "T"
}

func (p *Param) IsNull() bool {
	return p.Package == "" && p.Type == "" && p.Name == ""
}

func (p *Param) InMainPkg() bool {
	return p.Package == "main"
}

func (p *Param) IsTime() bool {
	return p.Package == "time" && p.Type == "Time"
}

func (p *Param) SetName(name string) {
	p.Name = name
}

func (p *Param) AllowType() bool {
	switch p.Type {
	case "string", "bytes":
		return true
	case "int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64":
		return true
	case "float64", "float32":
		return true
	case "bool":
		return true
	case "time.Time":
		return true
	default:
		return false
	}
}

func (p *Param) astGetParamType(param *ast.Field) {
	switch v := param.Type.(type) {
	case *ast.Ident:
		p.Type = v.Name
		if v.Obj != nil {
			p.Package = "UNDEFINED" // set a placeholder
		}
	case *ast.SelectorExpr:
		p.astGetEltType(v)
	case *ast.ArrayType:
		p.astGetEltType(v.Elt)
		p.IsArray = true
	case *ast.Ellipsis:
		p.astGetEltType(v.Elt)
		p.IsArray = true
	}
}

func (p *Param) astGetEltType(expr ast.Expr) string {
	switch v := expr.(type) {
	case *ast.Ident:
		p.Type = v.Name
		if v.Obj != nil {
			p.Package = "UNDEFINED"
		}
	case *ast.SelectorExpr:
		temp := new(Param)
		p.Type = v.Sel.Name
		p.Package = temp.astGetEltType(v.X)
	}
	return p.Type
}