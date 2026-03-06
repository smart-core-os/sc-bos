// protoc-gen-wrapper is a protoc plugin that generates a wrapper for each service that exists in the proto files.
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

//go:embed wrapper.go.gotxt
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
		filename := fmt.Sprintf("pkg/trait/%s/%s_wrap.pb.go", pkg, strings.ToLower(name))
		importPath := protogen.GoImportPath(fmt.Sprintf("github.com/smart-core-os/sc-golang/pkg/trait/%s", pkg))
		if *usePaths {
			qual := strings.ToLower(name)
			// avoid filenames like some/path/metadata_metadataapi_wrap.pb.go,
			// which would come from {prefix=some/path/metadata}{servicename=metadataapi}_wrap.pb.go.
			// Instead we want metadata_api_wrap.pb.go
			qual = strings.TrimPrefix(qual, strings.ReplaceAll(filepath.Base(file.GeneratedFilenamePrefix), "_", ""))
			if len(qual) > 0 {
				qual = "_" + qual
			}
			filename = file.GeneratedFilenamePrefix + qual + "_wrap.pb.go"
			importPath = file.GoImportPath
			pkg = string(file.GoPackageName)
		}

		g := plugin.NewGeneratedFile(filename, importPath)
		model := newServiceModel(g, service, file, pkg, name)
		err := serviceTmpl.Execute(g, model)
		if err != nil {
			return err
		}
	}
	return nil
}

func newServiceModel(g *protogen.GeneratedFile, service *protogen.Service, file *protogen.File, pkg, underlying string) ServiceModel {
	model := ServiceModel{
		Service: service,

		PackageName: pkg,
		Wrapper:     ident(underlying + "Wrapper"),
		Underlying:  ident(underlying),

		QualifiedServerName: g.QualifiedGoIdent(protogen.GoIdent{
			GoName:       service.GoName + "Server",
			GoImportPath: file.GoImportPath,
		}),
		ClientName: service.GoName + "Client",
		QualifiedClientName: g.QualifiedGoIdent(protogen.GoIdent{
			GoName:       service.GoName + "Client",
			GoImportPath: file.GoImportPath,
		}),
		QualifiedClientConstructor: g.QualifiedGoIdent(protogen.GoIdent{
			GoName:       "New" + service.GoName + "Client",
			GoImportPath: file.GoImportPath,
		}),
		QualifiedServiceDesc: g.QualifiedGoIdent(protogen.GoIdent{
			GoName:       service.GoName + "_ServiceDesc",
			GoImportPath: file.GoImportPath,
		}),

		WrapServerToClient: g.QualifiedGoIdent(protogen.GoIdent{
			GoName:       "ServerToClient",
			GoImportPath: "github.com/smart-core-os/sc-golang/pkg/wrap",
		}),
		GRPCClientConnInterface: g.QualifiedGoIdent(protogen.GoIdent{
			GoName:       "ClientConnInterface",
			GoImportPath: "google.golang.org/grpc",
		}),
		GRPCServiceDesc: g.QualifiedGoIdent(protogen.GoIdent{
			GoName:       "ServiceDesc",
			GoImportPath: "google.golang.org/grpc",
		}),
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
	Wrapper     Ident
	Underlying  Ident

	QualifiedServerName        string
	ClientName                 string
	QualifiedClientName        string
	QualifiedClientConstructor string
	QualifiedServiceDesc       string

	WrapServerToClient      string
	GRPCClientConnInterface string
	GRPCServiceDesc         string
}

type Ident struct {
	Exported string
	Private  string
}

func ident(n string) Ident {
	first, rest := n[:1], n[1:]
	return Ident{
		Exported: strings.ToUpper(first) + rest,
		Private:  strings.ToLower(first) + rest,
	}
}
