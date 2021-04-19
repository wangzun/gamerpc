/*
 *
 * Copyright 2020 gameRpc authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package main

import (
	"bytes"
	"fmt"
	"google.golang.org/protobuf/compiler/protogen"
	"os"
	"strings"
	"text/template"
)

var index int
var max int

func generate(gen *protogen.Plugin, file *protogen.File) *protogen.GeneratedFile {
	if len(file.Services) == 0 {
		return nil
	}
	prefix := strings.ReplaceAll(file.GeneratedFilenamePrefix,string(file.GoPackageName),*packageName)
	filename := prefix + "." + *packageName + ".go"
	importPath := strings.ReplaceAll(string(file.GoImportPath),string(file.GoPackageName),*packageName)
	g := gen.NewGeneratedFile(filename, protogen.GoImportPath(importPath))
	all := buildAllData(file,256)
	b := &bytes.Buffer{}
	tmpl, _ := template.New(filename).Parse(ServerFile)
	_ = tmpl.Execute(b, all)
	g.P(b.String())
	return g
}


func buildAllData(f *protogen.File,MaxIndex int) *All {
	all := &All{Ser:make([]*Service, MaxIndex)}
	goPackageName := strings.ReplaceAll(string(f.GoPackageName), " ", "")
	goImportPath := strings.ReplaceAll(string(f.GoImportPath), " ", "")
	generatedFilenamePrefix := strings.ReplaceAll(string(f.GeneratedFilenamePrefix), " ", "")
	if goPackageName != "" {
		all.GoPackageName = goPackageName
	}
	if goImportPath != "" {
		all.GoImportPath = goImportPath
	}
	if generatedFilenamePrefix  != "" {
		all.GeneratedFilenamePrefix = generatedFilenamePrefix
	}
	for _, e := range f.Services {
		scomment := string(e.Comments.Leading)
		scomment = strings.ReplaceAll(scomment, "\n", "")
		scomment = strings.ReplaceAll(scomment, " ", "")
		serviceName := e.GoName
		serviceName = strings.ReplaceAll(serviceName, " ", "")
		if serviceName == "" {
			continue
		}
		if len(serviceName) > max {
			max = len(serviceName)
		}
		cmd := index
		mes := make([]*Method, MaxIndex)
		ser := &Service{Name: serviceName, Cmd: cmd, Methods: mes, Comment: scomment}
		if all.Ser[cmd] != nil {
			panic(fmt.Sprintf("CMD 重复 %d", cmd))
		}
		all.Ser[cmd] = ser
		for i, m := range e.Methods {
			lcomment := string(m.Comments.Leading)
			mcomment := lcomment + " " + string(m.Comments.Trailing)
			mcomment = strings.ReplaceAll(mcomment, "\n", "")
			mcomment = strings.ReplaceAll(mcomment, " ", "")
			methodName := m.GoName
			if len(serviceName+"_"+methodName) > max {
				max = len(serviceName + "_" + methodName)
			}
			inPutName := string(m.Desc.Input().Name())
			outPUtName := string(m.Desc.Output().Name())
			act := i
			var typeId int
			var Param string
			if inPutName != "NULL" && outPUtName != "NULL" {
				typeId = 0
				Param = inPutName
			}else if inPutName != "NULL" && outPUtName == "NULL" {
				typeId = 1
				Param = inPutName
			}else if inPutName == "NULL" && outPUtName != "NULL" {
				typeId = 2
				Param = outPUtName
			}

			met := &Method{Param:Param,Typ: typeId, Comment: mcomment, Name: methodName, Input: inPutName, Output: outPUtName, Act: act, Service: serviceName, Cmd: cmd}
			if mes[act] != nil {
				panic(fmt.Sprintf("ACT 重复 %d", act))
			}
			mes[act] = met
		}
		index ++
	}
	newAll := &All{
		Ser:make([]*Service,0),
		GoImportPath:all.GoImportPath,
		GoPackageName:all.GoPackageName,
		GeneratedFilenamePrefix:all.GeneratedFilenamePrefix,
		Path:*packagePath,
		PackageName:*packageName,
	}
	for _,v := range all.Ser {
		if v != nil {
			newSer := &Service{Name:v.Name,Cmd:v.Cmd,Comment:v.Comment,Methods:make([]*Method,0)}
			for _, m := range v.Methods {
				if m != nil {
					newSer.Methods = append(newSer.Methods,m)
				}
			}
			newAll.Ser = append(newAll.Ser,newSer)
		}
	}
	return newAll
}

type All struct {
	Ser []*Service
	FileName string
	GoPackageName string
	GoImportPath string
	GeneratedFilenamePrefix string
	Path string
	PackageName string
}

type Service struct {
	Name    string
	Cmd     int
	Methods []*Method
	Comment string
}

type Method struct {
	Name    string
	Input   string
	Output  string
	Act     int
	Service string
	Cmd     int
	Comment string
	Typ     int
	Param   string
}

var ServerFile string = `// Code generated by gamerpc. DO NOT EDIT.
package {{.PackageName}}

import (
	"context"
	"errors"
    pb "{{.Path}}/{{.GoImportPath}}"
)

const (
    CALL int = 0
    CAST int = 1
)


// MethodDesc represents an GameRPC service's method specification.
type MethodDesc struct {
	MethodName  string
	CallHandler callHandler
	CastHandler castHandler
    Type        int
    Act         uint8
}

// ServiceDesc represents an GameRPC service's specification.
type ServiceDesc struct {
	ServiceName string
	Methods     []MethodDesc
    Cmd         uint8
	HandlerType interface{}
}

type Router struct {
    Url string
    Cmd uint8
    Act uint8
}

type ServiceRegistrar interface {
	RegisterService(desc *ServiceDesc, impl interface{})
}

type ClientConnInterface interface {
    Call(ctx context.Context,router *Router,in interface{}, out interface{}, opts ...interface{}) error
    Cast(ctx context.Context,router *Router,in interface{}, opts ...interface{}) error
}


type callHandler func(srv interface{}, ctx context.Context, dec func(interface{}) error) (interface{}, error)
type castHandler func(srv interface{}, ctx context.Context, dec func(interface{}) error) (error)

{{ range .Ser }}

//{{.Comment}}
type {{.Name}}Srv interface {
{{- range .Methods }}
    //{{.Comment}}
    {{- if .Typ}}
	{{.Name}}(context.Context, *pb.{{.Param}}) (error) 
    {{- else}}
	{{.Name}}(context.Context, *pb.{{.Input}}) (*pb.{{.Output}}, error)
    {{- end}}
{{- end }}
}

// Unim{{.Name}} must be embedded to have forward compatible implementations.
type Unim{{.Name}}Srv struct {
}

{{$name := .Name }}

{{- range .Methods }}
    //{{.Comment}}
    {{- if .Typ}}
func (*Unim{{print $name}}Srv) {{.Name}}(context.Context, *pb.{{.Param}}) (error) {
	return errors.New("method {{.Name}} not implemented")
}
    {{else}}
func (*Unim{{print $name}}Srv) {{.Name}}(context.Context, *pb.{{.Input}}) (*pb.{{.Output}},error) {
	return nil, errors.New("method {{.Name}} not implemented")
}
    {{- end}}
{{- end }}

{{- range .Methods }}

    {{- if .Typ}}
func _{{print $name}}_{{.Name}}_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error) (error) {
	in := new(pb.{{.Param}})
	if err := dec(in); err != nil {
		return err
	}
    return srv.({{print $name}}Srv).{{.Name}}(ctx, in)
}
    {{else}}
func _{{print $name}}_{{.Name}}_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error) (interface{}, error) {
	in := new(pb.{{.Input}})
	if err := dec(in); err != nil {
		return nil, err
	}
    return srv.({{print $name}}Srv).{{.Name}}(ctx, in)
}
    {{- end}}
{{- end }}

var {{.Name}}_ServiceDesc = ServiceDesc{
	ServiceName: "{{.Name}}",
	HandlerType: (*{{.Name}}Srv)(nil),
    Cmd:{{.Cmd}},
	Methods: []MethodDesc{
{{- range .Methods }}
		{
			MethodName: "{{.Name}}",
{{- if .Typ}}
			CastHandler:    _{{print $name}}_{{.Name}}_Handler,
            Type:CAST,
{{- else}}
			CallHandler:    _{{print $name}}_{{.Name}}_Handler,
            Type:CALL,
{{- end}}
            Act:{{.Act}},
		},
{{- end }}
	},
}

func Register{{.Name}}Srv(s ServiceRegistrar, srv {{.Name}}Srv) {
	s.RegisterService(&{{.Name}}_ServiceDesc, srv)
}

type {{.Name}}Cli interface {
{{- range .Methods }}
    //{{.Comment}}
    {{- if .Typ}}
	{{.Name}}(ctx context.Context,in *pb.{{.Param}},opts ...interface{}) (error) 
    {{- else}}
	{{.Name}}(ctx context.Context,in *pb.{{.Input}},opts ...interface{}) (*pb.{{.Output}}, error)
    {{- end}}
{{- end }}
}

type _{{.Name}}Cli struct {
	cc ClientConnInterface
}


func New{{.Name}}Cli(cc ClientConnInterface) {{.Name}}Cli {
	return &_{{.Name}}Cli{cc}
}

{{- range .Methods }}
{{- if .Typ}}
func (c *_{{print $name}}Cli) {{.Name}}(ctx context.Context, in *pb.{{.Param}}, opts ...interface{}) (error) {
    router := &Router{Url:"/{{print $name}}/{{.Name}}",Cmd:{{.Cmd}},Act:{{.Act}}}
	err := c.cc.Cast(ctx, router, in, opts...)
    return err
}
{{- else}}
func (c *_{{print $name}}Cli) {{.Name}}(ctx context.Context, in *pb.{{.Input}}, opts ...interface{}) (*pb.{{.Output}}, error) {
	out := new(pb.{{.Output}})
    router := &Router{Url:"/{{print $name}}/{{.Name}}",Cmd:{{.Cmd}},Act:{{.Act}}}
	err := c.cc.Call(ctx, router,in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}
{{- end}}
{{- end}}


{{ end }}
`

func generateGameRpcFile(all *All,fileName string,str string) {
	f,err:= os.OpenFile(fileName,os.O_WRONLY|os.O_CREATE|os.O_TRUNC,0666)
	if err != nil {
		panic(err)
	}
	tmpl, _ := template.New(fileName).Parse(ServerFile)
	_ = tmpl.Execute(f, all)

}



