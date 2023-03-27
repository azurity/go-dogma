package generator

import (
	"bytes"
	"fmt"
	"github.com/azurity/schema2code"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type GolangConfig struct {
	Path    string
	Package string
}

func formatGolangType(name string, packName string) string {
	if name == "null" {
		return "*Null"
	} else if name == "boolean" {
		return "bool"
	} else if name == "integer" {
		return "int"
	} else if name == "number" {
		return "float64"
	} else if name == "string" {
		return "string"
	} else {
		return packName + "." + formatName(name)
	}
}

var goFileGenerateLine = []byte("// Code generated by dogma. DO NOT EDIT.\n")

func RenderGolang(endpoints []Endpoint, types map[string]string, config *GolangConfig) error {
	for _, endpoint := range endpoints {
		endpointName := makeEndpointName(&endpoint)
		if endpoint.RetType != "" && endpoint.RetType[0] == '{' {
			types[endpointName+"Ret"] = endpoint.RetType
		}
	}

	os.RemoveAll(config.Path)
	os.Mkdir(config.Path, 0777)

	typeFile, err := os.OpenFile(filepath.Join(config.Path, "types.go"), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return err
	}
	typeFile.Write(goFileGenerateLine)

	buffer := &bytes.Buffer{}
	firstItem := true
	buffer.Write([]byte("{ \"$defs\":{"))
	for name, value := range types {
		if firstItem {
			firstItem = false
			buffer.Write([]byte("\n"))
		} else {
			buffer.Write([]byte(",\n"))
		}
		buffer.Write([]byte("\"" + name + "\":"))
		buffer.Write([]byte(value))
	}
	buffer.Write([]byte("\n}}"))

	err = schema2code.Generate(buffer, typeFile, &schema2code.GolangConfig{
		CommonConfig: schema2code.CommonConfig{RootType: ""},
		Package:      config.Package,
	})
	typeFile.Close()
	if err != nil {
		return err
	}

	packagePart := strings.Split(config.Package, "/")
	globalPackage := packagePart[len(packagePart)-1]

	packages := map[string]bool{}

	for _, endpoint := range endpoints {
		folderName := filepath.Join(append([]string{config.Path}, endpoint.Name[:len(endpoint.Name)-1]...)...)
		packages[strings.Join(endpoint.Name[:len(endpoint.Name)-1], "/")] = true
		packageName := endpoint.Name[len(endpoint.Name)-2]
		fileName := packageName + ".go"
		endpointName := makeEndpointName(&endpoint)

		os.MkdirAll(folderName, 0777)
		if _, err := os.Stat(filepath.Join(folderName, fileName)); os.IsNotExist(err) {
			file, _ := os.Create(filepath.Join(folderName, fileName))
			file.Write(goFileGenerateLine)
			file.Write([]byte(fmt.Sprintf("package %s\n\nimport \"%s\"\n\ntype Null = %s.Null\n\n", packageName, config.Package, globalPackage)))
			file.Close()
		}

		codeFile, _ := os.OpenFile(filepath.Join(folderName, fileName), os.O_RDWR|os.O_APPEND, 0666)

		//codeFile.Write([]byte(fmt.Sprintf()))
		codeFile.Write([]byte(fmt.Sprintf("type %sURLParam struct{\n", endpointName)))
		for _, item := range endpoint.Param {
			if item.Role == ParamRoleURL {
				codeFile.Write([]byte(fmt.Sprintf("\t%s %s `json:\"%s\"`\n", formatName(item.Name), formatGolangType(item.Type, globalPackage), item.Name)))
			}
		}
		codeFile.Write([]byte("}\n\n"))

		codeFile.Write([]byte(fmt.Sprintf("type %sCommonParam struct{\n", endpointName)))
		for _, item := range endpoint.Param {
			if item.Role == ParamRoleCommon {
				codeFile.Write([]byte(fmt.Sprintf("\t%s %s `json:\"%s\"`\n", formatName(item.Name), formatGolangType(item.Type, globalPackage), item.Name)))
			}
		}
		codeFile.Write([]byte("}\n\n"))

		//codeFile.Write([]byte(fmt.Sprintf("type %sParam struct{\n\t%sURLParam\n\t%sCommonParam\n}\n\n", endpointName, endpointName, endpointName)))

		retType := "*" + globalPackage + "." + formatName(endpoint.RetType)
		if endpoint.RetType == "" {
			retType = "*struct{}"
		} else if endpoint.RetType[0] == '{' {
			retType = "*" + globalPackage + "." + endpointName + "Ret"
		}
		codeFile.Write([]byte(fmt.Sprintf("type %s = func(urlParam %sURLParam, commonParam %sCommonParam) (%s, error)\n\n", endpointName, endpointName, endpointName, retType)))
		codeFile.Close()
	}

	descFile, _ := os.OpenFile(filepath.Join(config.Path, "desc.go"), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
	descFile.Write(goFileGenerateLine)

	descFile.Write([]byte(fmt.Sprintf("package %s\n\nimport (\n\t\"github.com/azurity/go-dogma\"\n\t\"reflect\"\n\n", globalPackage)))

	sortedPack := []string{}
	for name, _ := range packages {
		sortedPack = append(sortedPack, name)
	}
	sort.Strings(sortedPack)

	for _, name := range sortedPack {
		renamePart := strings.Split(name, "/")
		if len(renamePart) == 1 {
			descFile.Write([]byte(fmt.Sprintf("\t\"%s/%s\"\n", config.Package, name)))
			continue
		}
		for i, item := range renamePart {
			if i != 0 {
				renamePart[i] = formatName(item)
			}
		}
		rename := strings.Join(renamePart, "")
		descFile.Write([]byte(fmt.Sprintf("\t%s \"%s/%s\"\n", rename, config.Package, name)))
	}
	descFile.Write([]byte(")\n\n"))

	descFile.Write([]byte("var Desc = map[reflect.Type]dogma.Method{\n"))
	for _, endpoint := range endpoints {
		renamePart := append([]string{}, endpoint.Name[:len(endpoint.Name)-1]...)
		for i, item := range renamePart {
			if i != 0 {
				renamePart[i] = formatName(item)
			}
		}
		packRename := strings.Join(renamePart, "")
		endpointName := makeEndpointName(&endpoint)
		pathPart := append([]string{}, endpoint.Name...)

		for _, item := range endpoint.Param {
			if item.Role == ParamRoleURL {
				pathPart = append(pathPart, "{"+item.Name+"}")
			}
		}
		descFile.Write([]byte(fmt.Sprintf("\treflect.TypeOf(new(%s.%s)): {Name: \"/%s\", Method: \"%s\"},\n", packRename, endpointName, strings.Join(pathPart, "/"), endpoint.Method)))
	}
	descFile.Write([]byte("}\n\n"))

	descFile.Close()
	return nil
}
