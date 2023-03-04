package generator

import (
	"github.com/yuin/goldmark/ast"
	east "github.com/yuin/goldmark/extension/ast"
	"strings"
)

const (
	ParamRoleCommon int = iota
	ParamRoleURL
)

type Param struct {
	Name string
	Type string
	Role int
}

type Endpoint struct {
	Name    []string
	Method  string
	Param   []Param
	RetType string
}

type Header []string

func (h Header) Contains(aim string) bool {
	for _, item := range h {
		if item == aim {
			return true
		}
	}
	return false
}

func parseTable(node *east.Table, doc []byte) (Header, []map[string]string) {
	headers := []string{}
	ret := []map[string]string{}
	n := node.FirstChild()
	if n.Kind() != east.KindTableHeader {
		return headers, ret
	}
	for it := n.FirstChild(); it != nil; it = it.NextSibling() {
		headers = append(headers, string(it.Text(doc)))
	}
	for it := n.NextSibling(); it != nil; it = it.NextSibling() {
		row := map[string]string{}
		index := 0
		for cell := it.FirstChild(); cell != nil; cell = cell.NextSibling() {
			row[headers[index]] = string(cell.Text(doc))
			index += 1
		}
		ret = append(ret, row)
	}
	return headers, ret
}

func procParam(sec *section, source []byte) []Param {
	var table *east.Table
	for n := sec.node.NextSibling(); n != nil; n = n.NextSibling() {
		if n.Kind() == ast.KindHeading {
			break
		}
		if n.Kind() == east.KindTable {
			table = n.(*east.Table)
			break
		}
	}
	if table == nil {
		return []Param{}
	}
	header, body := parseTable(table, source)
	ret := []Param{}
	if header.Contains("Parameter") && header.Contains("Type") {
		for _, row := range body {
			param := Param{
				Name: row["Parameter"],
				Type: row["Type"],
				Role: ParamRoleCommon,
			}
			ret = append(ret, param)
		}
	} else {
		// TODO: log error
		return nil
	}
	return ret
}

func findFirstCodeBlock(node ast.Node) *ast.FencedCodeBlock {
	for n := node; n != nil; n = n.NextSibling() {
		if n.Kind() == ast.KindHeading {
			break
		}
		if n.Kind() == ast.KindFencedCodeBlock {
			return n.(*ast.FencedCodeBlock)
		}
	}
	return nil
}

func findFirstCode(node *ast.Paragraph) *ast.CodeSpan {
	n := node.FirstChild()
	for n != nil {
		if n.Kind() == ast.KindCodeSpan {
			return n.(*ast.CodeSpan)
		}
		n = n.FirstChild()
	}
	return nil
}

func procReturnValue(sec *section, source []byte) string {
	for n := sec.node.NextSibling(); n != nil; n = n.NextSibling() {
		if n.Kind() == ast.KindHeading {
			break
		}
		if n.Kind() == ast.KindFencedCodeBlock && string(n.(*ast.FencedCodeBlock).Info.Text(source)) == "json--schema" {
			lines := n.Lines()
			result := ""
			for i := 0; i < lines.Len(); i += 1 {
				item := lines.At(i)
				result += strings.TrimSpace(string(item.Value(source)))
			}
			return result
		}
		if n.Kind() == ast.KindParagraph {
			code := findFirstCode(n.(*ast.Paragraph))
			if code != nil {
				return string(code.Text(source))
			}
		}
	}
	return ""
}

func extractTypes(sec *section, source []byte) map[string]string {
	ret := map[string]string{}
	if sec.name != "Types" {
		for _, sub := range sec.subSections {
			subRet := extractTypes(sub, source)
			for name, definition := range subRet {
				// TODO: uni name check
				ret[name] = definition
			}
		}
	} else {
		for _, sub := range sec.subSections {
			code := findFirstCodeBlock(sub.node.NextSibling())
			if code != nil {
				lines := code.Lines()
				result := ""
				for i := 0; i < lines.Len(); i += 1 {
					item := lines.At(i)
					result += strings.TrimSpace(string(item.Value(source)))
				}
				ret[sub.name] = result
			}
		}
	}
	return ret
}
