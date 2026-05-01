package formatter

import (
	"fmt"
	"html"
	"net/url"
	"strings"

	"github.com/yuin/goldmark"
	gast "github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	east "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/text"
)

var githubMarkdown = goldmark.New(goldmark.WithExtensions(extension.GFM))

func githubMarkdownToTelegramHTML(markdown string) string {
	markdown = strings.TrimSpace(markdown)
	if markdown == "" {
		return ""
	}

	source := []byte(markdown)
	root := githubMarkdown.Parser().Parse(text.NewReader(source))
	renderer := markdownTelegramRenderer{source: source}
	renderer.render(root)

	return cleanMarkdownOutput(renderer.String())
}

type markdownTelegramRenderer struct {
	source []byte
	b      strings.Builder
}

func (r *markdownTelegramRenderer) String() string {
	return r.b.String()
}

func (r *markdownTelegramRenderer) render(node gast.Node) {
	switch n := node.(type) {
	case *gast.Document:
		r.renderBlockChildren(n)
	case *gast.Paragraph:
		r.renderInlineChildren(n)
		r.ensureNewline()
	case *gast.TextBlock:
		r.writeEscaped(string(n.Text(r.source)))
		r.ensureNewline()
	case *gast.Heading:
		r.write("<b>")
		r.renderInlineChildren(n)
		r.write("</b>")
		r.ensureNewline()
	case *gast.Text:
		r.writeEscaped(string(n.Value(r.source)))
		if n.HardLineBreak() {
			r.write("\n")
		} else if n.SoftLineBreak() {
			r.write(" ")
		}
	case *gast.String:
		r.writeEscaped(string(n.Value))
	case *gast.Emphasis:
		if n.Level >= 2 {
			r.wrap("b", n)
		} else {
			r.wrap("i", n)
		}
	case *gast.CodeSpan:
		r.write("<code>")
		r.writeEscaped(strings.TrimSpace(string(n.Text(r.source))))
		r.write("</code>")
	case *gast.FencedCodeBlock:
		r.writePre(string(n.Text(r.source)))
	case *gast.CodeBlock:
		r.writePre(string(n.Text(r.source)))
	case *gast.Link:
		r.renderLink(string(n.Destination), n)
	case *gast.AutoLink:
		r.renderAutoLink(n)
	case *gast.Image:
		r.renderImage(n)
	case *gast.Blockquote:
		inner := renderMarkdownSubtree(r.source, n)
		if inner != "" {
			r.write("<blockquote>")
			r.write(inner)
			r.write("</blockquote>")
			r.ensureNewline()
		}
	case *gast.List:
		r.renderList(n)
	case *gast.ListItem:
		r.renderBlockChildren(n)
	case *gast.ThematicBreak:
		r.write("-----")
		r.ensureNewline()
	case *gast.HTMLBlock:
		r.writeEscaped(string(n.Text(r.source)))
		r.ensureNewline()
	case *gast.RawHTML:
		r.writeEscaped(string(n.Text(r.source)))
	case *east.Strikethrough:
		r.wrap("s", n)
	case *east.TaskCheckBox:
		if n.IsChecked {
			r.write("[x] ")
		} else {
			r.write("[ ] ")
		}
	case *east.Table:
		r.renderBlockChildren(n)
	case *east.TableHeader:
		r.renderBlockChildren(n)
	case *east.TableRow:
		r.renderTableRow(n)
	case *east.TableCell:
		r.renderInlineChildren(n)
	default:
		if node.HasChildren() {
			r.renderChildren(node)
			return
		}
		r.writeEscaped(string(node.Text(r.source)))
	}
}

func (r *markdownTelegramRenderer) renderChildren(node gast.Node) {
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		r.render(child)
	}
}

func (r *markdownTelegramRenderer) renderInlineChildren(node gast.Node) {
	r.renderChildren(node)
}

func (r *markdownTelegramRenderer) renderBlockChildren(node gast.Node) {
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		r.render(child)
		if child.NextSibling() != nil {
			r.ensureBlankLine()
		}
	}
}

func (r *markdownTelegramRenderer) renderList(list *gast.List) {
	index := list.Start
	if index == 0 {
		index = 1
	}

	for item := list.FirstChild(); item != nil; item = item.NextSibling() {
		prefix := "- "
		if list.IsOrdered() {
			prefix = fmt.Sprintf("%d. ", index)
			index++
		}

		itemText := renderMarkdownSubtree(r.source, item)
		r.writeIndentedBlock(prefix, itemText)
	}
	r.ensureNewline()
}

func (r *markdownTelegramRenderer) renderTableRow(row gast.Node) {
	var cells []string
	for cell := row.FirstChild(); cell != nil; cell = cell.NextSibling() {
		cells = append(cells, renderMarkdownSubtree(r.source, cell))
	}
	r.write(strings.Join(cells, " | "))
	r.ensureNewline()
}

func (r *markdownTelegramRenderer) renderLink(destination string, node gast.Node) {
	if href, ok := safeTelegramURL(destination); ok {
		r.write(`<a href="`)
		r.write(escapeHTML(href))
		r.write(`">`)
		r.renderInlineChildren(node)
		r.write("</a>")
		return
	}
	r.renderInlineChildren(node)
}

func (r *markdownTelegramRenderer) renderAutoLink(node *gast.AutoLink) {
	label := string(node.Label(r.source))
	destination := string(node.URL(r.source))
	if node.AutoLinkType == gast.AutoLinkEmail && !strings.HasPrefix(strings.ToLower(destination), "mailto:") {
		destination = "mailto:" + destination
	}

	if href, ok := safeTelegramURL(destination); ok {
		r.write(`<a href="`)
		r.write(escapeHTML(href))
		r.write(`">`)
		r.writeEscaped(label)
		r.write("</a>")
		return
	}
	r.writeEscaped(label)
}

func (r *markdownTelegramRenderer) renderImage(node *gast.Image) {
	label := strings.TrimSpace(renderMarkdownSubtree(r.source, node))
	if label == "" {
		label = "image"
	}

	if href, ok := safeTelegramURL(string(node.Destination)); ok {
		r.write(`Image: <a href="`)
		r.write(escapeHTML(href))
		r.write(`">`)
		r.write(label)
		r.write("</a>")
		return
	}

	r.write("Image: ")
	r.write(label)
}

func (r *markdownTelegramRenderer) wrap(tag string, node gast.Node) {
	r.write("<")
	r.write(tag)
	r.write(">")
	r.renderInlineChildren(node)
	r.write("</")
	r.write(tag)
	r.write(">")
}

func (r *markdownTelegramRenderer) writePre(value string) {
	value = strings.TrimRight(value, "\n")
	if value == "" {
		return
	}
	r.write("<pre>")
	r.writeEscaped(value)
	r.write("</pre>")
	r.ensureNewline()
}

func (r *markdownTelegramRenderer) writeIndentedBlock(prefix string, value string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return
	}

	lines := strings.Split(value, "\n")
	r.write(prefix)
	r.write(lines[0])
	r.ensureNewline()
	for _, line := range lines[1:] {
		line = strings.TrimRight(line, " ")
		if line == "" {
			r.ensureNewline()
			continue
		}
		r.write("  ")
		r.write(line)
		r.ensureNewline()
	}
}

func (r *markdownTelegramRenderer) write(value string) {
	r.b.WriteString(value)
}

func (r *markdownTelegramRenderer) writeEscaped(value string) {
	r.write(escapeHTML(value))
}

func (r *markdownTelegramRenderer) ensureNewline() {
	if r.b.Len() == 0 {
		return
	}
	current := r.b.String()
	if strings.HasSuffix(current, "\n") {
		return
	}
	r.write("\n")
}

func (r *markdownTelegramRenderer) ensureBlankLine() {
	if r.b.Len() == 0 {
		return
	}
	current := r.b.String()
	if strings.HasSuffix(current, "\n\n") {
		return
	}
	if strings.HasSuffix(current, "\n") {
		r.write("\n")
		return
	}
	r.write("\n\n")
}

func renderMarkdownSubtree(source []byte, node gast.Node) string {
	renderer := markdownTelegramRenderer{source: source}
	renderer.renderChildren(node)
	return cleanMarkdownOutput(renderer.String())
}

func safeTelegramURL(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", false
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return "", false
	}

	switch strings.ToLower(parsed.Scheme) {
	case "http", "https", "mailto":
		return raw, true
	default:
		return "", false
	}
}

func cleanMarkdownOutput(value string) string {
	value = strings.TrimSpace(value)
	for strings.Contains(value, "\n\n\n") {
		value = strings.ReplaceAll(value, "\n\n\n", "\n\n")
	}
	return value
}

func escapeHTML(value string) string {
	return html.EscapeString(value)
}
