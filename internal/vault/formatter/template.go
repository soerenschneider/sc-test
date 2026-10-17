package formatter

import (
	"bytes"
	"text/template"
)

type TemplateFormatter struct {
	tmpl *template.Template
}

func NewTemplateFormatterFromTemplate(tmpl string) (*TemplateFormatter, error) {
	t, err := template.New("tmpl").Parse(tmpl)
	if err != nil {
		return nil, err
	}

	return &TemplateFormatter{
		tmpl: t,
	}, nil
}

func NewTemplateFormatterFromFile(file string) (*TemplateFormatter, error) {
	t, err := template.ParseFiles(file)
	if err != nil {
		return nil, err
	}

	return &TemplateFormatter{
		tmpl: t,
	}, nil
}

func (t *TemplateFormatter) Format(data map[string]string) ([]byte, error) {
	var result bytes.Buffer
	if err := t.tmpl.Execute(&result, data); err != nil {
		return nil, err
	}

	return result.Bytes(), nil
}
