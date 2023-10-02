package main

import (
	"fmt"
	"go/doc/comment"
	"strings"

	"github.com/yoheimuta/go-protoparser/v4/parser"
	"github.com/yoheimuta/go-protoparser/v4/parser/meta"
)

const (
	XDomainExtension         = "x-domain"
	XDomainCategoryExtension = "x-category"
)

// ProtoFile is a parsing unit
type ProtoFile struct {
	// a proto file could have multiple service
	Services []Service
	// a proto file should have multiple object
	Objects []Object
	// a proto file should have multiple enum
	Enums []Enum
}

// Service is a grpc service
type Service struct {
	// service only use Comment placed at the beginning
	Comment string
	// the package name of the proto file
	PackageName string
	// my name
	ServiceName string
	// a service has multiple endpoint
	Operations []Endpoint
}

// Endpoint is also called method or interface
type Endpoint struct {
	// the package name of the proto file
	PackageName string
	// the enclosing service name
	ServiceName string
	// my name
	MethodName string
	// the url path where api gateway resolves
	URLPath string
	// the http method, which is always POST
	HTTPMethod string
	// Comment placed at the beginning, as well as inline-Comment placed at the ending
	Comment string

	Markdown string

	Type     RPCType
	Request  Request
	Response Response

	Extensions map[string]string
}

type RPCType int

const (
	Unary = iota
	ServerStreaming
	ClientStreaming
	BidirectionalStreaming
)

type Request struct {
	Params []Field
	Type   string
}

type Response struct {
	Params []Field
	Type   string
}

type Field struct {
	// Comment placed at the beginning, as well as inline-Comment placed at the ending
	Comment     string
	Name        string
	TypeName    string
	KeyTypeName string
	Repeat      bool
	// the Enclosing type name of this field
	Enclosing string
	// reference to the enclosing proto file
	protoFile *ProtoFile
}

// Object is user-defined field type
type Object struct {
	// only use Comment placed at the beginning
	Comment string
	// refers by field.typ
	Name string
	// attributes
	Attrs []Field
}

// Enum is user-defined type which has one of a pre-defined list of values
type Enum struct {
	// only use Comment placed at the beginning
	Comment string
	// refers by field.typ
	Name string
	// Constants
	Constants []EnumField
}

type EnumField struct {
	// Comment placed at the beginning, as well as inline-Comment placed at the ending
	Comment string
	Name    string
	Val     string
	// the Enclosing type name of this field
	Enclosing string
}

func (t RPCType) String() string {
	switch t {
	case Unary:
		return "unary"
	case ClientStreaming:
		return "client-streaming"
	case ServerStreaming:
		return "server-streaming"
	case BidirectionalStreaming:
		return "bidirectional-streaming"
	}
	return "unknown"
}

func (pf *ProtoFile) ComposeFrom(pp *parser.Proto) error {
	comments := make([]*parser.Comment, 0)
	for _, x := range pp.ProtoBody {
		if comment, ok := x.(*parser.Comment); ok {
			comments = append(comments, comment)
		}
		if pack, ok := x.(*parser.Package); ok {
			comments = append(comments, pack.Comments...)
		}
	}
	_, protobufFileExtensions := composeHeadComment(comments, nil)

	// find all services in proto body
	for _, x := range pp.ProtoBody {
		if service, ok := x.(*parser.Service); ok {
			if err := pf.addService(service, pp, protobufFileExtensions); err != nil {
				return fmt.Errorf("add service: %w", err)
			}
		}
	}
	pf.addObjectsAndEnums(pp)
	return nil
}

func (pf *ProtoFile) addService(
	ps *parser.Service,
	pp *parser.Proto,
	protobufFileExtensions map[string]string,
) error {
	var s Service
	var serviceExtensions map[string]string
	s.Comment, serviceExtensions = composeHeadComment(ps.Comments, protobufFileExtensions)
	s.PackageName = extractPackageName(pp)
	s.ServiceName = ps.ServiceName
	var err error
	s.Operations, err = pf.composeInterfaces(s, ps, pp, serviceExtensions)
	if err != nil {
		return fmt.Errorf("compose interfaces: %w", err)
	}

	pf.Services = append(pf.Services, s)
	return nil
}

func extractComment(pc *parser.Comment) string {
	if pc == nil {
		return ""
	}
	lines := removeInitialEmptyLinesAndAsterisks(pc.Lines())
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func removeInitialEmptyLinesAndAsterisks(lines []string) []string {
	res := make([]string, 0)
	hasNotEmpty := false
	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		if !hasNotEmpty && (trimmedLine == "" ||
			strings.HasPrefix(line, "* ") ||
			trimmedLine == "*" ||
			strings.HasPrefix(line, "*\t")) {
			continue
		}
		hasNotEmpty = true
		res = append(res, removeInitialAsterisk(line))
	}
	return res
}

func composeHeadComment(pcs []*parser.Comment, parentExtensions map[string]string) (string, map[string]string) {
	extensions := make(map[string]string)
	for k, v := range parentExtensions {
		extensions[k] = v
	}
	ss := make([]string, 0, len(pcs))
	for _, pc := range pcs {
		if newExtensions, ok := getExtensions(pc); ok {
			for key, val := range newExtensions {
				extensions[key] = val
			}
			continue
		}
		s := extractComment(pc)
		ss = append(ss, s)
	}
	return strings.Join(ss, "\n"), extensions
}

func getExtensions(pc *parser.Comment) (map[string]string, bool) {
	extensions := make(map[string]string, 0)
	for _, line := range removeInitialEmptyLinesAndAsterisks(pc.Lines()) {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Domain:") {
			extensions[XDomainExtension] = strings.TrimSpace(strings.TrimPrefix(line, "Domain:"))
		}
		if strings.HasPrefix(line, "Category:") {
			extensions[XDomainCategoryExtension] = strings.TrimSpace(strings.TrimPrefix(line, "Category:"))
		}
		if strings.HasPrefix(line, XDomainExtension+":") {
			extensions[XDomainExtension] = strings.TrimSpace(strings.TrimPrefix(line, XDomainExtension+":"))
		}
		if strings.HasPrefix(line, XDomainCategoryExtension+":") {
			extensions[XDomainCategoryExtension] = strings.TrimSpace(strings.TrimPrefix(line, XDomainCategoryExtension+":"))
		}
	}
	if len(extensions) > 0 {
		return extensions, true
	}
	return nil, false
}

func removeInitialAsterisk(line string) string {
	line = strings.TrimSpace(line)
	if strings.HasPrefix(line, "* ") {
		return strings.TrimPrefix(line, "* ")
	}
	if line == "*" {
		return ""
	}
	return strings.TrimPrefix(line, "*\t")
}

func composeHeadAndInlineComment(
	pcs []*parser.Comment,
	pic *parser.Comment,
	sep string,
	parentExtensions map[string]string,
) (string, map[string]string) {
	head, extensions := composeHeadComment(pcs, parentExtensions)
	inline := extractComment(pic)
	if head == "" {
		return inline, extensions
	}
	if inline == "" {
		return head, extensions
	}
	return head + sep + inline, extensions
}

func extractPackageName(pp *parser.Proto) string {
	for _, x := range pp.ProtoBody {
		if p, ok := x.(*parser.Package); ok {
			return p.Name
		}
	}
	return "(missed-package)"
}

func (pf *ProtoFile) composeInterfaces(
	s Service,
	ps *parser.Service,
	pp *parser.Proto,
	parentExtensions map[string]string,
) ([]Endpoint, error) {
	eps := make([]Endpoint, 0, len(ps.ServiceBody))
	for _, x := range ps.ServiceBody {
		var op Endpoint
		op.PackageName = s.PackageName
		op.ServiceName = s.ServiceName
		if rpc, ok := x.(*parser.RPC); ok {
			op.MethodName = rpc.RPCName
			op.Comment, op.Extensions = composeHeadAndInlineComment(rpc.Comments, rpc.InlineComment, "\n", parentExtensions)
			op.Type = extractRPCType(rpc)
			var err error
			if op.Request, err = extractRPCRequest(rpc.RPCRequest, pp, pf); err != nil {
				return nil, fmt.Errorf("extract RPC request: %w", err)
			}
			if op.Response, err = extractRPCResponse(rpc.RPCResponse, pp, pf); err != nil {
				return nil, fmt.Errorf("extract RPC response: %w", err)
			}
		}
		op.validate()
		eps = append(eps, op)
	}
	return eps, nil
}

func (e *Endpoint) OperationMarkdown() string {
	return e.Markdown
}

func (e *Endpoint) validate() {
	e.URLPath = "/" + e.PackageName + "/" + e.ServiceName + "/" + e.MethodName
	e.HTTPMethod = "POST"
	if e.Type != Unary {
		e.HTTPMethod = "GET" // websocket uses GET
	}
}

func (e *Endpoint) XDomainCategory() string {
	return e.Extensions[XDomainCategoryExtension]
}

func (e *Endpoint) XDomain() string {
	return e.Extensions[XDomainExtension]
}

func (e *Endpoint) OperationID() string {
	return e.MethodName
}

func (e *Endpoint) FullName() string {
	return fmt.Sprintf("%s.%s/%s", e.PackageName, e.ServiceName, e.MethodName)
}

func (e *Endpoint) Summary() *string {
	summary := e.Comment
	indexEOL := strings.IndexByte(summary, '\n')
	if indexEOL > 0 {
		summary = summary[:indexEOL]
	}
	return &summary
}

func (e *Endpoint) Description() *string {
	summary := e.Comment
	indexEOL := strings.IndexByte(summary, '\n')
	if indexEOL > 0 {
		summary = summary[indexEOL+1:]
	}
	return &summary
}

func extractRPCType(rpc *parser.RPC) RPCType {
	if rpc.RPCRequest.IsStream && rpc.RPCResponse.IsStream {
		return BidirectionalStreaming
	}
	if rpc.RPCRequest.IsStream {
		return ClientStreaming
	}
	if rpc.RPCResponse.IsStream {
		return ServerStreaming
	}
	return Unary
}

func extractRPCRequest(rr *parser.RPCRequest, pp *parser.Proto, pf *ProtoFile) (r Request, err error) {
	msg, err := findMessage(pp, rr.MessageType)
	if err != nil {
		return r, fmt.Errorf("find message: %w", err)
	}
	r.Params = composeFields(msg, msg.MessageName, pf)
	r.Type = rr.MessageType
	return r, nil
}

func extractRPCResponse(rr *parser.RPCResponse, pp *parser.Proto, pf *ProtoFile) (r Response, err error) {
	msg, err := findMessage(pp, rr.MessageType)
	if err != nil {
		return r, fmt.Errorf("find message: %w", err)
	}
	r.Params = composeFields(msg, msg.MessageName, pf)
	r.Type = rr.MessageType
	return r, nil
}

func findMessage(pp *parser.Proto, mt string) (*parser.Message, error) {
	for _, x := range pp.ProtoBody {
		if m, ok := x.(*parser.Message); ok && m.MessageName == mt {
			return m, nil
		}
	}
	if mt == "google.protobuf.Empty" {
		return &parser.Message{
			MessageName:                  "google.protobuf.Empty",
			MessageBody:                  nil,
			Comments:                     nil,
			InlineComment:                nil,
			InlineCommentBehindLeftCurly: nil,
			Meta:                         meta.Meta{},
		}, nil
	}

	return nil, fmt.Errorf("proto doesn't has message %q", mt)
}

func composeFields(pm *parser.Message, enclosing string, protoFile *ProtoFile) []Field {
	fs := make([]Field, 0, len(pm.MessageBody))
	for _, x := range pm.MessageBody {
		if pf, ok := x.(*parser.Field); ok {
			var f Field
			f.Comment, _ = composeHeadAndInlineComment(pf.Comments, pf.InlineComment, " ", nil)
			f.Name = pf.FieldName
			f.TypeName = pf.Type
			f.Repeat = pf.IsRepeated
			f.Enclosing = enclosing
			f.protoFile = protoFile
			fs = append(fs, f)
		}
		if pf, ok := x.(*parser.MapField); ok {
			var f Field
			f.Comment, _ = composeHeadAndInlineComment(pf.Comments, pf.InlineComment, " ", nil)
			f.Name = pf.MapName
			f.TypeName = pf.Type
			f.KeyTypeName = pf.KeyType
			f.Repeat = false
			f.Enclosing = enclosing
			f.protoFile = protoFile
			fs = append(fs, f)
		}
	}
	return fs
}

var scalarTypes = map[string]struct{}{
	"double":   {},
	"float":    {},
	"int32":    {},
	"int64":    {},
	"uint32":   {},
	"uint64":   {},
	"sint32":   {},
	"sint64":   {},
	"fixed32":  {},
	"fixed64":  {},
	"sfixed32": {},
	"sfixed64": {},
	"bool":     {},
	"string":   {},
	"bytes":    {},
}

func (f Field) isScalar() (typename string, ok bool) {
	if f.KeyTypeName != "" {
		return "", false
	}
	_, ok = scalarTypes[f.TypeName]
	return f.TypeName, ok
}

func (f Field) isEnum() (typename string, ok bool) {
	if f.KeyTypeName != "" {
		return "", false
	}
	scopes := strings.Split(f.Enclosing, ".")
	for i := len(scopes); i >= 0; i-- {
		scope := strings.Join(scopes[:i], ".")
		qualified := f.TypeName
		if scope != "" {
			qualified = scope + "." + f.TypeName
		}
		for _, e := range f.protoFile.Enums {
			if qualified == e.Name {
				return e.Name, true
			}
		}
	}
	return "", false
}

func (f Field) isObject() (typename string, ok bool) {
	if f.KeyTypeName != "" {
		return "", false
	}
	scopes := strings.Split(f.Enclosing, ".")
	for i := len(scopes); i >= 0; i-- {
		scope := strings.Join(scopes[:i], ".")
		qualified := f.TypeName
		if scope != "" {
			qualified = scope + "." + f.TypeName
		}
		for _, o := range f.protoFile.Objects {
			if qualified == o.Name {
				return o.Name, true
			}
		}
	}
	return "", false
}

func (f Field) isMap() (typeNameKey string, typeNameValue string, ok bool) {
	if f.KeyTypeName == "" {
		return "", "", false
	}
	scopes := strings.Split(f.Enclosing, ".")
	for i := len(scopes); i >= 0; i-- {
		scope := strings.Join(scopes[:i], ".")
		var ok bool
		if typeNameKey, ok = f.findQualified(scope, f.KeyTypeName); ok {
			if typeNameValue, ok = f.findQualified(scope, f.TypeName); ok {
				return typeNameKey, typeNameValue, true
			}
		}
	}
	return f.KeyTypeName, f.TypeName, true
}

func (f Field) findQualified(scope string, qualified string) (string, bool) {
	if scope != "" {
		qualified = scope + "." + qualified
	}
	for _, o := range f.protoFile.Objects {
		if qualified == o.Name {
			return qualified, true
		}
	}
	return "", false
}

// Type resolves f's type to a readable format in the scope of protoFile
func (f Field) Type() (r string) {
	if f.TypeName == "" {
		return "(nil)"
	}
	r = f.TypeBase()
	if f.Repeat {
		r = "[]" + r
	}
	return r
}

func (f Field) TypeBase() string {
	if typeName, ok := f.isScalar(); ok {
		return typeName
	}
	if typeNameEnum, ok := f.isEnum(); ok {
		return "enum " + typeNameEnum
	}
	if typeNameObject, ok := f.isObject(); ok {
		return "message " + typeNameObject
	}
	if typeNameKeyObject, typeNameValueObject, ok := f.isMap(); ok {
		return "map <" + typeNameKeyObject + ", " + typeNameValueObject + ">"
	}
	return "(" + f.TypeName + ")"
}

// convert typename to href id
func href(typename string) string {
	return strings.Join(strings.Split(strings.ToLower(typename), "."), "")
}

// add all messages and enums in the proto, but exclude the messages used directly by interfaces
func (pf *ProtoFile) addObjectsAndEnums(pp *parser.Proto) {
	excludes := make(map[string]bool)
	for _, s := range pf.Services {
		for _, op := range s.Operations {
			excludes[op.Request.Type] = true
			excludes[op.Response.Type] = true
		}
	}

	var extractMessagesAndEnums func(body []parser.Visitee, enclosingName string)
	extractMessagesAndEnums = func(body []parser.Visitee, enclosingName string) {
		for _, x := range body {
			if enum, ok := x.(*parser.Enum); ok {
				e := composeEnum(enum, enclosingName)
				pf.Enums = append(pf.Enums, e)
			} else if msg, ok := x.(*parser.Message); ok {
				o := composeObject(msg, enclosingName, pf)
				if !excludes[o.Name] {
					pf.Objects = append(pf.Objects, o)
					// continue
				}

				extractMessagesAndEnums(msg.MessageBody, enclosingName+msg.MessageName+".")
			}
		}
	}

	extractMessagesAndEnums(pp.ProtoBody, ".")
}

func composeObject(pm *parser.Message, enclosingName string, pf *ProtoFile) (o Object) {
	enclosingName = enclosingName[1:] // trim the first "."
	o.Comment, _ = composeHeadComment(pm.Comments, nil)
	o.Name = enclosingName + pm.MessageName
	o.Attrs = composeFields(pm, o.Name, pf)
	return o
}

func composeEnum(pe *parser.Enum, enclosingName string) (e Enum) {
	enclosingName = enclosingName[1:] // trim the first "."
	e.Comment, _ = composeHeadComment(pe.Comments, nil)
	e.Name = enclosingName + pe.EnumName
	e.Constants = composeEnumFields(pe, e.Name)
	return e
}

func composeEnumFields(pe *parser.Enum, enclosing string) []EnumField {
	fs := make([]EnumField, 0, len(pe.EnumBody))
	for _, x := range pe.EnumBody {
		if pf, ok := x.(*parser.EnumField); ok {
			var f EnumField
			f.Comment, _ = composeHeadAndInlineComment(pf.Comments, pf.InlineComment, " ", nil)
			f.Name = pf.Ident
			f.Val = pf.Number
			f.Enclosing = enclosing
			fs = append(fs, f)
		}
	}
	return fs
}

func (e *Endpoint) EmptySummary() bool {
	summary := e.Summary()
	return summary == nil || strings.TrimSpace(*summary) == ""
}

func (e *Endpoint) DescriptionMarkdown() string {
	description := e.Description()
	if description == nil {
		return ""
	}
	var p comment.Parser
	doc := p.Parse(*description)
	var pr comment.Printer
	docBytes := pr.Markdown(doc)
	return string(docBytes)
}

func (e *Endpoint) Typei18n() string {
	typeStr := `**Unary RPCs** (*клиент делает запрос на сервер в виде` +
		` одного сообщения и получает ответ в виде одного сообщения от сервера*)`
	switch e.Type {
	case ClientStreaming:
		typeStr = `**ClientStreaming RPCs** (*клиент делает запрос на сервер` +
			` в виде последовательности сообщений и получает ответ в виде одного сообщения от сервера*)`
	case ServerStreaming:
		typeStr = `**ServerStreaming RPCs** (*клиент делает запрос на сервер` +
			` в виде одного сообщения и получает ответ в виде последовательности сообщений от сервера*)`
	case BidirectionalStreaming:
		typeStr = `**BidirectionalStreaming RPCs** (*клиент делает запрос на сервер` +
			` в виде последовательности сообщений и получает ответ в виде последовательности сообщений от сервера*)`
	}
	return typeStr
}

// TypeHRef is Type in addition to a href
func (f Field) TypeHRef() (r string) {
	if f.TypeName == "" {
		return "(nil)"
	}
	r = f.typeHRefBase()
	if f.Repeat {
		r = "[]" + r
	}
	return r
}

func (f Field) typeHRefBase() string {
	if typeName, ok := f.isScalar(); ok {
		return typeName
	}

	if typeNameEnum, ok := f.isEnum(); ok {
		return "[enum " + typeNameEnum + "](#enum-" + href(typeNameEnum) + ")"
	}

	if typeNameObject, ok := f.isObject(); ok {
		return "[" + typeNameObject + "](#тип-" + href(typeNameObject) + ")"
	}

	if typeNameKeyObject, typeNameValueObject, ok := f.isMap(); ok {
		keyTypeString := typeNameKeyObject
		if _, ok := scalarTypes[typeNameKeyObject]; !ok {
			keyTypeString = "[" + typeNameKeyObject + "](#тип-" + href(typeNameKeyObject) + ")"
		}
		valueTypeString := typeNameValueObject
		if _, ok := scalarTypes[typeNameValueObject]; !ok {
			valueTypeString = "[" + typeNameValueObject + "](#тип-" + href(typeNameValueObject) + ")"
		}
		return "map<" + keyTypeString + ", " + valueTypeString + ">"
	}

	return "(" + f.TypeName + ")"
}
