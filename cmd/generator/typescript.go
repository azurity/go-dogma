package generator

import (
	"bytes"
	_ "embed"
	"fmt"
	"github.com/azurity/schema2code"
	"os"
	"sort"
	"strings"
)

//go:embed handle_ts
var handleTs []byte

type TypescriptConfig struct {
	FilePath string
}

func formatTypescriptType(name string, packName string) string {
	if name == "null" {
		return "null"
	} else if name == "boolean" {
		return "boolean"
	} else if name == "integer" {
		return "number"
	} else if name == "number" {
		return "number"
	} else if name == "string" {
		return "string"
	} else {
		return packName + "." + formatName(name)
	}
}

var typescriptFileGenerateLine = []byte("/* tslint:disable */\n")

func RenderTypescript(endpoints []Endpoint, types map[string]string, config *TypescriptConfig) error {
	for _, endpoint := range endpoints {
		endpointName := makeEndpointName(&endpoint)
		if endpoint.RetType != "" && endpoint.RetType[0] == '{' {
			types[endpointName+"Ret"] = endpoint.RetType
		}
	}

	typeFile, err := os.OpenFile(config.FilePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return err
	}
	typeFile.Write(typescriptFileGenerateLine)

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

	typeFile.Write([]byte("export namespace types {\n"))
	err = schema2code.Generate(buffer, typeFile, &schema2code.TypescriptConfig{
		CommonConfig: schema2code.CommonConfig{RootType: ""},
	})
	typeFile.Write([]byte("}\n\n"))
	defer typeFile.Close()
	if err != nil {
		return err
	}

	//packages := map[string]bool{}
	packages := map[string][]Endpoint{}

	for _, endpoint := range endpoints {
		renamePart := append([]string{}, endpoint.Name[:len(endpoint.Name)-1]...)
		for i, item := range renamePart {
			renamePart[i] = formatName(item)
		}
		packName := strings.Join(renamePart, ".")
		if _, ok := packages[packName]; !ok {
			packages[packName] = []Endpoint{}
		}
		packages[packName] = append(packages[packName], endpoint)
	}

	sortedPackages := []string{}
	for name, _ := range packages {
		sortedPackages = append(sortedPackages, name)
	}
	sort.Strings(sortedPackages)

	for _, name := range sortedPackages {
		typeFile.Write([]byte(fmt.Sprintf("export namespace %s {\n", name)))

		for _, endpoint := range packages[name] {
			endpointName := makeEndpointName(&endpoint)
			typeFile.Write([]byte(fmt.Sprintf("    export interface %sURLParam {\n", endpointName)))
			for _, item := range endpoint.Param {
				if item.Role == ParamRoleURL {
					typeFile.Write([]byte(fmt.Sprintf("        \"%s\": %s;\n", item.Name, formatTypescriptType(item.Type, "types"))))
				}
			}
			typeFile.Write([]byte("    }\n\n"))

			typeFile.Write([]byte(fmt.Sprintf("    export function %sURLParam$check(obj: %sURLParam) {\n", endpointName, endpointName)))
			for _, item := range endpoint.Param {
				if item.Role == ParamRoleURL {
					if formatTypescriptType(item.Type, "types") == "types."+formatName(item.Type) {
						typeFile.Write([]byte(fmt.Sprintf("        types.$check(\"%s\", obj[\"%s\"]);\n", formatName(item.Type), item.Name)))
					}
				}
			}
			typeFile.Write([]byte("    }\n\n"))

			typeFile.Write([]byte(fmt.Sprintf("    export interface %sCommonParam {\n", endpointName)))
			for _, item := range endpoint.Param {
				if item.Role == ParamRoleCommon {
					typeFile.Write([]byte(fmt.Sprintf("        \"%s\": %s;\n", item.Name, formatTypescriptType(item.Type, "types"))))
				}
			}
			typeFile.Write([]byte("    }\n\n"))

			typeFile.Write([]byte(fmt.Sprintf("    export function %sCommonParam$check(obj: %sCommonParam) {\n", endpointName, endpointName)))
			for _, item := range endpoint.Param {
				if item.Role == ParamRoleCommon {
					if formatTypescriptType(item.Type, "types") == "types."+formatName(item.Type) {
						typeFile.Write([]byte(fmt.Sprintf("        types.$check(\"%s\", obj[\"%s\"]);\n", formatName(item.Type), item.Name)))
					}
				}
			}
			typeFile.Write([]byte("    }\n\n"))

			retType := "types." + formatName(endpoint.RetType)
			if endpoint.RetType == "" {
				retType = "void"
			} else if endpoint.RetType[0] == '{' {
				retType = "types." + endpointName + "Ret"
			}
			typeFile.Write([]byte(fmt.Sprintf("    export type %s = (urlParam: %sURLParam, commonParam: %sCommonParam) => %s;\n\n", endpointName, endpointName, endpointName, retType)))

			typeFile.Write([]byte(fmt.Sprintf("    export function %sRet$check(obj: %s) {", endpointName, retType)))
			if retType != "void" {
				typeFile.Write([]byte(fmt.Sprintf(" types.$check(\"%s\", obj); ", retType[6:])))
			}
			typeFile.Write([]byte("}\n\n"))
		}

		typeFile.Write([]byte("}\n\n"))
	}

	typeFile.Write([]byte("export interface $TypeMap {\n"))
	for _, endpoint := range endpoints {
		renamePart := append([]string{}, endpoint.Name[:len(endpoint.Name)-1]...)
		for i, item := range renamePart {
			renamePart[i] = formatName(item)
		}
		packName := strings.Join(renamePart, ".")
		endpointName := makeEndpointName(&endpoint)
		pathPart := append([]string{}, endpoint.Name...)

		for _, item := range endpoint.Param {
			if item.Role == ParamRoleURL {
				pathPart = append(pathPart, "{"+item.Name+"}")
			}
		}
		typeFile.Write([]byte(fmt.Sprintf("    \"%s.%s\": %s.%s;\n", packName, endpointName, packName, endpointName)))
	}
	typeFile.Write([]byte("}\n\n"))

	typeFile.Write([]byte("export const $Desc = {\n"))
	for _, endpoint := range endpoints {
		renamePart := append([]string{}, endpoint.Name[:len(endpoint.Name)-1]...)
		for i, item := range renamePart {
			renamePart[i] = formatName(item)
		}
		packName := strings.Join(renamePart, ".")
		endpointName := makeEndpointName(&endpoint)
		pathPart := append([]string{}, endpoint.Name...)
		pathTemplatePart := append([]string{}, endpoint.Name...)

		for _, item := range endpoint.Param {
			if item.Role == ParamRoleURL {
				pathPart = append(pathPart, "{"+item.Name+"}")
				pathTemplatePart = append(pathTemplatePart, "${param."+item.Name+"}")
			}
		}

		pathTemplate := fmt.Sprintf("(param: %s.%sURLParam) => `/%s`", packName, endpointName, strings.Join(pathPart, "/"))

		fullName := packName + "." + endpointName
		metaData := fmt.Sprintf("{ name: \"%s\", method: \"%s\", urlFunc: %s, urlParamCheck: %sURLParam$check, commonParamCheck: %sCommonParam$check, retCheck: %sRet$check }",
			strings.Join(pathPart, "/"), endpoint.Method, pathTemplate, fullName, fullName, fullName)

		typeFile.Write([]byte(fmt.Sprintf("    \"%s.%s\": %s,\n", packName, endpointName, metaData)))
	}
	typeFile.Write([]byte("}\n\n"))
	typeFile.Write(handleTs)
	return nil
}
