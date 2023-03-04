package generator

import (
	"github.com/yuin/goldmark/ast"
	"strings"
)

func procRequest(sec *section, source []byte) *Endpoint {
	for n := sec.node.NextSibling(); n != nil; n = n.NextSibling() {
		if n.Kind() == ast.KindHeading {
			break
		}
		if n.Kind() == ast.KindParagraph && n.FirstChild() != nil && n.FirstChild().Kind() == ast.KindCodeSpan {
			reqText := string(n.FirstChild().Text(source))
			reqPart := strings.Split(reqText, " ")
			if len(reqPart) != 2 {
				return nil
			}
			method := reqPart[0]
			// TODO: check method
			namePart := strings.Split(reqPart[1], "/")
			if namePart[0][:4] == "http" {
				namePart = namePart[3:]
			}
			index := 0
			for ; index < len(namePart); index += 1 {
				item := namePart[index]
				if item[0] == '<' && item[len(item)-1] == '>' {
					break
				}
				// TODO: check legal
			}
			reqName := namePart[:index]
			params := []Param{}
			for ; index < len(namePart); index += 1 {
				item := namePart[index]
				if item[0] != '<' || item[len(item)-1] != '>' {
					// TODO: log error
					return nil
				}
				params = append(params, Param{
					Name: item[1 : len(item)-1],
					Type: "string",
					Role: ParamRoleURL,
				})
			}
			return &Endpoint{
				Name:    reqName,
				Method:  method,
				Param:   params,
				RetType: "",
			}
		}
	}
	return nil
}

func extractRestApi(rootSec *section, source []byte) []Endpoint {
	if rootSec.notApi {
		return nil
	}
	var requestSec *section
	var paramSec *section
	for _, sec := range rootSec.subSections {
		if sec.name == "HTTP Request" {
			if requestSec != nil {
				return nil
			}
			requestSec = sec
		} else if sec.name == "Parameters" {
			if paramSec != nil {
				return nil
			}
			paramSec = sec
		}
	}
	if requestSec == nil {
		ret := []Endpoint{}
		for _, sub := range rootSec.subSections {
			ret = append(ret, extractRestApi(sub, source)...)
		}
		return ret
	} else {
		endpoint := procRequest(requestSec, source)
		if endpoint == nil {
			return nil
		}
		if paramSec != nil {
			param := procParam(paramSec, source)
			if param == nil {
				return nil
			}
			endpoint.Param = append(endpoint.Param, param...)
		}
		retType := procReturnValue(rootSec, source)
		endpoint.RetType = retType
		endpoint.Title = formatTitle(string(rootSec.node.Text(source)))
		return []Endpoint{*endpoint}
	}
	return nil
}
