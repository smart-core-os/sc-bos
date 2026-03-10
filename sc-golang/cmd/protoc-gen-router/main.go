// protoc-gen-router is a protoc plugin that generates a router for each service that exists in the proto files.
//
// This outputs to the {out}/pkg/trait/{trait name} package unless usePaths=true option is specified.
package main

import (
	_ "embed"
	"flag"
	"fmt"
	"path/filepath"
	"strings"
	"text/template"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/types/pluginpb"
)

//go:embed router.go.gotxt
var serviceTmplStr string
var serviceTmpl *template.Template

var (
	flags    flag.FlagSet
	usePaths = flags.Bool("usePaths", false, "use paths option instead of hard-coded pkg/trait/{trait} output")
)

func main() {
	var err error
	serviceTmpl, err = template.New("service").Parse(serviceTmplStr)
	if err != nil {
		panic(err)
	}

	opts := protogen.Options{
		ParamFunc: flags.Set,
	}
	opts.Run(func(plugin *protogen.Plugin) error {
		for _, file := range plugin.Files {
			if !file.Generate {
				continue
			}
			if err := generateFile(plugin, file); err != nil {
				return err
			}
		}
		plugin.SupportedFeatures = uint64(pluginpb.CodeGeneratorResponse_FEATURE_PROTO3_OPTIONAL)
		return nil
	})
}

func generateFile(plugin *protogen.Plugin, file *protogen.File) error {
	pkg := filepath.Base(filepath.Dir(file.GeneratedFilenamePrefix))
	// looked like something/something/traits/something, use something more sensible
	if pkg == "traits" {
		pkg = filepath.Base(file.GeneratedFilenamePrefix)
	}
	pkg = strings.ReplaceAll(pkg, "_", "")
	if !strings.HasSuffix(pkg, "pb") {
		pkg += "pb"
	}

	for _, service := range file.Services {
		name := trimPrefixIgnoreCase(service.GoName, strings.TrimSuffix(pkg, "pb"))
		filename := fmt.Sprintf("pkg/trait/%s/%s_router.pb.go", pkg, strings.ToLower(name))
		importPath := protogen.GoImportPath(fmt.Sprintf("github.com/smart-core-os/sc-golang/pkg/trait/%s", pkg))
		if *usePaths {
			qual := strings.ToLower(name)
			// avoid filenames like some/path/metadata_metadataapi_router.pb.go,
			// which would come from {prefix=some/path/metadata}{servicename=metadataapi}_router.pb.go.
			// Instead we want metadata_api_router.pb.go
			qual = strings.TrimPrefix(qual, strings.ReplaceAll(filepath.Base(file.GeneratedFilenamePrefix), "_", ""))
			if len(qual) > 0 {
				qual = "_" + qual
			}
			filename = file.GeneratedFilenamePrefix + qual + "_router.pb.go"
			importPath = file.GoImportPath
			pkg = string(file.GoPackageName)
		}
		routerName := name + "Router"

		g := plugin.NewGeneratedFile(filename, importPath)
		model := newServiceModel(g, service, file, pkg, routerName)
		err := serviceTmpl.Execute(g, model)
		if err != nil {
			return err
		}
	}
	return nil
}

func newServiceModel(g *protogen.GeneratedFile, service *protogen.Service, file *protogen.File, pkg, routerName string) ServiceModel {
	// imports required by all
	g.QualifiedGoIdent(protogen.GoIdent{GoImportPath: "google.golang.org/grpc"})
	g.QualifiedGoIdent(protogen.GoIdent{GoImportPath: "github.com/smart-core-os/sc-golang/pkg/router"})
	g.QualifiedGoIdent(protogen.GoIdent{GoImportPath: "fmt"})

	model := ServiceModel{
		Service: service,
		ServerName: ident(g, protogen.GoIdent{
			GoName:       service.GoName + "Server",
			GoImportPath: file.GoImportPath,
		}),
		ClientName: ident(g, protogen.GoIdent{
			GoName:       service.GoName + "Client",
			GoImportPath: file.GoImportPath,
		}),
		UnimplementedServerName: ident(g, protogen.GoIdent{
			GoName:       "Unimplemented" + service.GoName + "Server",
			GoImportPath: file.GoImportPath,
		}),
		RegisterService: ident(g, protogen.GoIdent{
			GoName:       "Register" + service.GoName + "Server",
			GoImportPath: file.GoImportPath,
		}),
		PackageName: pkg,
		RouterName:  routerName,
	}

	for _, method := range service.Methods {
		// make sure the correct imports are available
		g.QualifiedGoIdent(protogen.GoIdent{
			GoName:       "Context",
			GoImportPath: "context",
		})
		if method.Desc.IsStreamingServer() {
			g.QualifiedGoIdent(protogen.GoIdent{
				GoName:       "EOF",
				GoImportPath: "io",
			})
		}

		model.Methods = append(model.Methods, ServiceMethod{
			Method:    method,
			Streaming: method.Desc.IsStreamingServer(),
			ServerStream: ident(g, protogen.GoIdent{
				GoName:       fmt.Sprintf("%s_%sServer", service.GoName, method.GoName),
				GoImportPath: file.GoImportPath,
			}),
			GoInput:  ident(g, method.Input.GoIdent),
			GoOutput: ident(g, method.Output.GoIdent),
		})
	}
	return model
}

func trimPrefixIgnoreCase(s, prefix string) string {
	ls, lp := strings.ToLower(s), strings.ToLower(prefix)
	ls = strings.TrimPrefix(ls, lp)
	return s[len(s)-len(ls):]
}

type ServiceModel struct {
	*protogen.Service

	PackageName string
	RouterName  string

	ServerName              Ident
	ClientName              Ident
	UnimplementedServerName Ident
	RegisterService         Ident

	Methods []ServiceMethod
}

type ServiceMethod struct {
	*protogen.Method

	Streaming    bool
	ServerStream Ident

	GoInput  Ident
	GoOutput Ident
}

type Ident struct {
	Exported  string
	Private   string
	Qualified string
}

func ident(g *protogen.GeneratedFile, n protogen.GoIdent) Ident {
	first, rest := n.GoName[:1], n.GoName[1:]
	return Ident{
		Exported:  strings.ToUpper(first) + rest,
		Private:   strings.ToLower(first) + rest,
		Qualified: g.QualifiedGoIdent(n),
	}
}
