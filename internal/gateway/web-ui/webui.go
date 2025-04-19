package webui

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/tekig/clerk/internal/logger"
	"github.com/tekig/clerk/internal/pb"
	"github.com/tekig/clerk/internal/searcher"
	"github.com/tekig/clerk/internal/uuid"
)

type WebUI struct {
	searcher *searcher.Searcher
	server   *echo.Echo
	style    *chroma.Style
	formater *html.Formatter
	template *template.Template
	address  string
}

type SearcherConfig struct {
	Searcher *searcher.Searcher
	Address  string
}

//go:embed static/*
var static embed.FS

func NewWebUI(config SearcherConfig) (*WebUI, error) {
	s := styles.Get("xcode-dark")
	if s == nil {
		s = styles.Fallback
	}

	g := &WebUI{
		searcher: config.Searcher,
		server:   echo.New(),
		style:    s,
		template: template.New(""),
		formater: html.New(html.WithClasses(true)),
		address:  config.Address,
	}

	tmplFuncs := map[string]any{
		"print_id":   g.printID,
		"pretty":     g.pretty,
		"pretty_css": g.prettyCSS,
	}

	g.template.Funcs(tmplFuncs)
	template.Must(g.template.ParseFS(static, "static/*.html"))

	g.server.Use(
		middleware.Recover(),
		logger.EchoLogger(),
	)

	g.server.GET("/", g.Index)
	g.server.GET("/event/:id", g.Search)

	return g, nil
}

func (g *WebUI) Index(c echo.Context) error {
	if err := g.template.ExecuteTemplate(c.Response(), "index", nil); err != nil {
		return fmt.Errorf("template: %w", err)
	}

	return nil
}

func (g *WebUI) Search(c echo.Context) error {
	event, err := g.search(c)
	if err != nil {
		if err := g.template.ExecuteTemplate(c.Response(), "error", err.Error()); err != nil {
			return fmt.Errorf("template error: %w", err)
		}

		return nil
	}

	if err := g.template.ExecuteTemplate(c.Response(), "event", event); err != nil {
		return fmt.Errorf("template event: %w", err)
	}

	return nil
}

func (g *WebUI) search(c echo.Context) (*pb.Event, error) {
	id, err := uuid.FromString(c.Param("id"))
	if err != nil {
		return nil, fmt.Errorf("id from string: %w", err)
	}

	event, err := g.searcher.Search(c.Request().Context(), id)
	if err != nil {
		return nil, fmt.Errorf("search: %w", err)
	}

	return event, nil
}

func (g *WebUI) Run() error {
	return g.server.Start(g.address)
}

func (g *WebUI) Shutdown() error {
	return g.server.Shutdown(context.Background())
}

func (g *WebUI) printID(id []byte) string {
	if id == nil {
		return ""
	}

	return uuid.UUID(id).String()
}

func (g *WebUI) prettyCSS() template.CSS {
	buf := bytes.NewBuffer(nil)

	if err := g.formater.WriteCSS(buf, g.style); err != nil {
		buf.WriteString(fmt.Errorf("write css: %w", err).Error())
	}

	return template.CSS(buf.String())
}

func (g *WebUI) pretty(v any) template.HTML {
	var raw string
	switch v := v.(type) {
	case *pb.Attribute_AsString:
		raw = v.AsString
	case *pb.Attribute_AsInt64:
		raw = fmt.Sprintf("%d", v.AsInt64)
	case *pb.Attribute_AsDouble:
		raw = fmt.Sprintf("%v", v.AsDouble)
	case *pb.Attribute_AsBool:
		raw = fmt.Sprintf("%v", v.AsBool)
	default:
		raw = fmt.Sprintf("%v", v)
	}

	text, err := g.highlight(raw)
	if err != nil {
		text = fmt.Errorf("highlight: %w", err).Error()
	}

	return template.HTML(text)
}

func (g *WebUI) highlight(text string) (string, error) {
	var l = lexers.Fallback
	buf := bytes.NewBuffer(nil)
	if err := json.Indent(buf, []byte(text), "", "\t"); err == nil {
		text = buf.String()
		l = lexers.Get("json")
	}
	buf.Reset()

	it, err := l.Tokenise(nil, text)
	if err != nil {
		return "", fmt.Errorf("tokinise: %w", err)
	}

	if err := g.formater.Format(buf, g.style, it); err != nil {
		return "", fmt.Errorf("format: %w", err)
	}

	return buf.String(), nil
}
