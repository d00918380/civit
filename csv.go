package main

import (
	_ "embed"
	"html/template"
	"io"

	"github.com/d00918380/civit/internal/trpc"
)

//go:embed csv.csv
var csvTemplate string

func csv(w io.Writer, items []*trpc.Item) error {
	data := &data{
		Items: items,
	}
	funcs := template.FuncMap{}
	t, err := template.New("csv.csv").Funcs(funcs).Parse(csvTemplate)
	if err != nil {
		return err
	}

	return t.Execute(w, data)
}
