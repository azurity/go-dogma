package main

import (
	"flag"
	"github.com/azurity/go-dogma/cmd/generator"
	"log"
	"os"
)

func main() {
	doc := flag.String("doc", "", "document file path")
	golangPackage := flag.String("golang-package", "", "golang package")
	golangPath := flag.String("golang-path", "", "golang output folder")
	typescriptPath := flag.String("typescript-path", "", "typescript output file")
	flag.Parse()

	if *doc == "" {
		log.Panicln("need input a doc")
	}

	golang := false
	if *golangPackage != "" && *golangPath != "" {
		golang = true
	}
	typescript := false
	if *typescriptPath != "" {
		typescript = true
	}

	if !golang {
		log.Panicln("must have at least one output")
	}

	data, _ := os.ReadFile(*doc)
	endpoints, types := generator.ParseDocument(data)

	if golang {
		err := generator.RenderGolang(endpoints, types, &generator.GolangConfig{Package: *golangPackage, Path: *golangPath})
		if err != nil {
			log.Println(err)
		}
	}
	if typescript {
		err := generator.RenderTypescript(endpoints, types, &generator.TypescriptConfig{FilePath: *typescriptPath})
		if err != nil {
			log.Println(err)
		}
	}
}
