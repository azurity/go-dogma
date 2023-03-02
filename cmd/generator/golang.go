package generator

import (
	"bytes"
	"fmt"
	"github.com/azurity/schema2code"
	"os"
	"path/filepath"
	"strings"
)

type GolangConfig struct {
	Path    string
	Package string
}

func formatName(name string) string {
	snake := strings.ReplaceAll(name, "-", "_")
	if snake == "" {
		return snake
	}
	return strings.ToUpper(snake[:1]) + snake[1:]
}

func RenderGolang(endpoints []Endpoint, types map[string]string, config *GolangConfig) error {
	os.Remove(config.Path)
	os.Mkdir(config.Path, 0777)

	typeFile, err := os.OpenFile(filepath.Join(config.Path, "types.go"), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return err
	}

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
		endpointName := formatName(endpoint.Name[len(endpoint.Name)-1]) + endpoint.Method

		os.MkdirAll(folderName, 0777)
		if _, err := os.Stat(filepath.Join(folderName, fileName)); os.IsNotExist(err) {
			file, _ := os.Create(filepath.Join(folderName, fileName))
			file.Write([]byte(fmt.Sprintf("package %s\n\nimport \"%s\"\n\ntype Dummy = %s.Null", packageName, config.Package, globalPackage)))
			file.Close()
		}

		codeFile, _ := os.OpenFile(filepath.Join(folderName, fileName), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)

		//codeFile.Write([]byte(fmt.Sprintf()))
		codeFile.Write([]byte(fmt.Sprintf("type %sURLParam struct{\n", endpointName)))
		for _, item := range endpoint.Param {
			if item.Role == ParamRoleURL {
				codeFile.Write([]byte(fmt.Sprintf("\t%s %s `json::\"%s\"`\n", formatName(item.Name), formatName(item.Type), item.Name)))
			}
		}
		codeFile.Write([]byte("}\n\n"))

		codeFile.Write([]byte(fmt.Sprintf("type %sCommonParam struct{\n", endpointName)))
		for _, item := range endpoint.Param {
			if item.Role == ParamRoleCommon {
				codeFile.Write([]byte(fmt.Sprintf("\t%s %s `json:\"%s\"`\n", formatName(item.Name), formatName(item.Type), item.Name)))
			}
		}
		codeFile.Write([]byte("}\n\n"))

		//codeFile.Write([]byte(fmt.Sprintf("type %sParam struct{\n\t%sURLParam\n\t%sCommonParam\n}\n\n", endpointName, endpointName, endpointName)))

		codeFile.Write([]byte(fmt.Sprintf("type %s = func(urlParam %sURLParam, commonParam %sCommonParam) (%s, error)\n\n", endpointName, endpointName, endpointName, formatName(endpoint.RetType))))
		codeFile.Close()
	}

	descFile, _ := os.OpenFile(filepath.Join(config.Path, "desc.go"), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)

	descFile.Write([]byte(fmt.Sprintf("package %s\n\nimport (\n\"reflect\"\n\"github.com/azurity/go-dogma\"\n\n")))
	for name, _ := range packages {
		rename := strings.Join(strings.Split(name, "/"), "_")
		descFile.Write([]byte(fmt.Sprintf("\t%s \"%s/%s\"\n", rename, config.Package, name)))
	}
	descFile.Write([]byte(")\n\n"))

	descFile.Write([]byte("var desc = map[*reflect.Type]dogma.Method{\n"))
	for _, endpoint := range endpoints {
		packRename := strings.Join(endpoint.Name[:len(endpoint.Name)-1], "_")
		endpointName := formatName(endpoint.Name[len(endpoint.Name)-1])
		pathPart := append([]string{}, endpoint.Name...)

		for _, item := range endpoint.Param {
			if item.Role == ParamRoleURL {
				pathPart = append(pathPart, "{"+item.Name+"}")
			}
		}
		descFile.Write([]byte(fmt.Sprintf("\treflect.TypeOf(%s.%s): {Name: \"/%s\", Method: \"%s\"},\n", packRename, endpointName, strings.Join(pathPart, "/"), endpoint.Method)))
	}
	descFile.Write([]byte("}\n\n"))

	descFile.Close()
	return nil
}
