package main

import (
	"bytes"
	"log"
	"strings"
	"text/template"
)

const ProtoTpl = `
{{- /* ------------------------------------------------------------- */ -}}
{{define "service"}}
## Сервис {{.ServiceName}}

{{.Comment}}
{{range .Operations}}
{{template "operation" .}}
{{end}}
{{end}}

{{- /* ------------------------------------------------------------- */ -}}
{{define "operation"}}
# {{.FullName}}
[[^]](#описание-api)


{{- if not .EmptySummary}}
{{.Summary}}
{{- end}}

**Тип:** {{.Typei18n}}


{{.DescriptionMarkdown}}

{{if .Request.Empty}}Запрос не содержит параметров
{{else}}Параметры запроса:
{{template "fields" .Request.Params}}{{end}}
{{if .Response.Empty}}Ответ не содержит данных
{{else}}Параметры ответа:
{{template "fields" .Response.Params}}{{end}}


{{end}}

{{- /* ------------------------------------------------------------- */ -}}

{{define "enum"}}
### Enum {{.Name}}
[[^]](#описание-api)

{{.Comment}}

Константы

|   Значение   |   Наименование    |  Описание |
| --------- | --------- | ------------ |
{{- range .Constants}}
| {{.Val}}  | {{.Name}} | {{.Comment}} |
{{- end}}
{{end}}

{{- /* ------------------------------------------------------------- */ -}}

{{- define "object"}}
### Тип {{.Name}}
[[^]](#описание-api)

{{.Comment}}

{{if .Empty}}Тип не содержит аттрибутов
{{else}}Аттрибуты:
{{template "fields" .Attrs}}{{end}}
{{- end}}

{{- /* ------------------------------------------------------------- */ -}}

{{define "fields"}}
|   Название    |   Тип параметра    |  Описание |
| --------- | --------- | ------------ |
{{- range .}}
| {{.Name}} | {{.TypeHRef}} | {{.Comment}} |
{{- end}}
{{end}}

{{- /* ------------------------------------------------------------- */ -}}

{{define "toc"}}
{{- range .Services}}
* [Сервис {{.ServiceName}}]({{.HRef}})
{{- range .Operations}}
    * [{{.MethodName}}]({{.HRef}})
{{- end}}
{{- end}}
* [Типы](#objects)
{{- range .Objects}}
    * [Тип {{.Name}}]({{.HRef}})
{{- end}}
* [Enums](#enums)
{{- range .Enums}}
    * [Enum {{.Name}}]({{.HRef}})
{{- end}}
{{end}}

{{- /* ------------------------------------------------------------- */ -}}

# Описание API

{{template "toc" .}}

{{range .Services}}
{{template "service" .}}
{{end}}

{{- if .Objects}}
## Типы
{{- range .Objects}}
    {{template "object" .}}
{{- end}}
{{- end}}

{{- if .Enums}}
## Перечисления
    {{- range .Enums}}
        {{template "enum" .}}
    {{- end}}
{{- end}}
`

func (pf ProtoFile) GenerateMarkdown() string {
	proto := template.Must(template.New("proto").Parse(ProtoTpl))
	buf := bytes.Buffer{}
	if err := proto.Execute(&buf, pf); err != nil {
		log.Panicf("failed to execute template: %v", err)
	}
	return buf.String()
}

// HRef generates a cross reference ID used in markdown
func (s Service) HRef() string {
	return "#сервис-" + strings.ToLower(s.ServiceName)
}

func headerToMDLink(h string) string {
	res := strings.ToLower(h)
	//res = strings.ReplaceAll(res, "_", "")
	res = strings.ReplaceAll(res, ".", "")
	res = strings.ReplaceAll(res, ",", "")
	res = strings.ReplaceAll(res, "-", "")
	res = strings.ReplaceAll(res, "/", "")
	res = strings.ReplaceAll(res, "\\", "")
	res = strings.ReplaceAll(res, ":", "")
	res = strings.ReplaceAll(res, ":", "")
	res = strings.ReplaceAll(res, " ", "-")
	return "#" + res
}

func (e Endpoint) HRef() string {
	return headerToMDLink(e.FullName())
}

func (o Object) HRef() string {
	return "#тип-" + href(o.Name)
}

func (e Enum) HRef() string {
	return "#enum-" + href(e.Name)
}

func (r Request) Empty() bool {
	return len(r.Params) == 0
}

func (r Response) Empty() bool {
	return len(r.Params) == 0
}

func (o Object) Empty() bool {
	return len(o.Attrs) == 0
}

func (e Endpoint) IsWebSocket() bool {
	return e.Type != Unary
}
