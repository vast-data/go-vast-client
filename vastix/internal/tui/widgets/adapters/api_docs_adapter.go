package adapters

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/getkin/kin-openapi/openapi3"
	"vastix/internal/colors"
	"vastix/internal/database"
	"vastix/internal/tui/widgets/common"
	api "github.com/vast-data/go-vast-client/openapi_schema"
)

// ---------------------------------------------------------------------------
// Item types
// ---------------------------------------------------------------------------

type docItemKind int

const (
	docItemSeparator docItemKind = iota
	docItemSection               // [METHOD] /path/  summary
	docItemParam                 // - name [type]
)

type docItem struct {
	kind        docItemKind
	display     string // compact rendered text (plain, no cursor styling)
	name        string // param/field name
	typeName    string // param type
	required    bool
	description string
	in          string // "query", "body", "path"
	depth       int    // indentation level (0 = top-level)
	children    []docItem
	expanded    bool
	isArrayItem bool // true when this item is an element of an array schema
}

// ---------------------------------------------------------------------------
// ApiDocsAdapter
// ---------------------------------------------------------------------------

// ApiDocsAdapter renders a scrollable, cursor-driven view of the OpenAPI
// documentation for the current resource.
type ApiDocsAdapter struct {
	db           *database.Service
	resourcePath string // e.g. "topics"

	items      []docItem
	selectIdx  []int // indices into items[] of selectable (param) rows
	cursor     int   // index into selectIdx

	viewport      viewport.Model
	ready         bool
	width, height int
}

// NewApiDocsAdapter creates the adapter. Call Load() when the resource path is known.
func NewApiDocsAdapter(db *database.Service) *ApiDocsAdapter {
	return &ApiDocsAdapter{db: db}
}

// Load (re-)builds the items list for a resource.
func (a *ApiDocsAdapter) Load(resourcePath string) {
	a.resourcePath = resourcePath
	a.items = nil
	a.selectIdx = nil
	a.cursor = 0
	a.ready = false

	a.buildItems()
}

// buildItems fetches swagger paths and constructs the flat item list.
func (a *ApiDocsAdapter) buildItems() {
	allPaths, err := api.GetAllPaths()
	if err != nil {
		a.items = []docItem{{kind: docItemSection, display: fmt.Sprintf("Error loading API schema: %v", err)}}
		return
	}

	prefix := "/" + a.resourcePath + "/"
	prefixNoSlash := "/" + a.resourcePath

	// Collect and sort paths that belong to this resource
	var matchedPaths []string
	for path := range allPaths {
		if path == prefix || path == prefixNoSlash ||
			strings.HasPrefix(path, prefix) {
			matchedPaths = append(matchedPaths, path)
		}
	}
	sort.Strings(matchedPaths)

	if len(matchedPaths) == 0 {
		a.items = []docItem{{
			kind:    docItemSection,
			display: fmt.Sprintf("No API paths found for /%s/", a.resourcePath),
		}}
		return
	}

	methodOrder := map[string]int{"GET": 0, "POST": 1, "PUT": 2, "PATCH": 3, "DELETE": 4, "HEAD": 5, "OPTIONS": 6}

	first := true
	for _, path := range matchedPaths {
		methods := allPaths[path]

		// Sort methods in logical order
		sort.Slice(methods, func(i, j int) bool {
			oi, oj := methodOrder[methods[i]], methodOrder[methods[j]]
			return oi < oj
		})

		for _, method := range methods {
			if !first {
				a.items = append(a.items, docItem{kind: docItemSeparator})
			}
			first = false

			summary, _ := api.GetOperationSummary(method, path)
			a.items = append(a.items, docItem{
				kind:    docItemSection,
				display: formatSection(method, path, summary),
			})

			// Query parameters
			params, _ := api.GetQueryParameters(method, path)
			if len(params) > 0 {
				a.items = append(a.items, docItem{
					kind:    docItemSeparator,
					display: "  parameters:",
				})
			for _, p := range params {
				item := paramToItem(p)
				item.depth = 1
				a.selectIdx = append(a.selectIdx, len(a.items))
				a.items = append(a.items, item)
			}
			}

			// Request body
			bodySchema, err := api.GetRequestBodySchema(method, path)
			if err == nil && bodySchema != nil && bodySchema.Value != nil {
				a.items = append(a.items, docItem{
					kind:    docItemSeparator,
					display: "  body:",
				})
				required := map[string]bool{}
				for _, r := range bodySchema.Value.Required {
					required[r] = true
				}
				// Collect and sort properties for deterministic order
				type propEntry struct {
					name string
					ref  *openapi3.SchemaRef
				}
				var props []propEntry
				for name, ref := range bodySchema.Value.Properties {
					props = append(props, propEntry{name, ref})
				}
				sort.Slice(props, func(i, j int) bool { return props[i].name < props[j].name })

			for _, prop := range props {
				item := schemaToItemAtDepth(prop.name, prop.ref, required[prop.name], "body", 1)
				a.selectIdx = append(a.selectIdx, len(a.items))
				a.items = append(a.items, item)
			}
			}
		}
	}
}

// ---------------------------------------------------------------------------
// Navigation
// ---------------------------------------------------------------------------

func (a *ApiDocsAdapter) cursorUp() {
	if a.cursor > 0 {
		a.cursor--
		a.scrollToCursor()
	}
}

func (a *ApiDocsAdapter) cursorDown() {
	if a.cursor < len(a.selectIdx)-1 {
		a.cursor++
		a.scrollToCursor()
	}
}

func (a *ApiDocsAdapter) TogglePopup() {
	if len(a.selectIdx) == 0 {
		return
	}
	idx := a.selectIdx[a.cursor]
	item := a.items[idx]
	if len(item.children) > 0 {
		a.toggleExpand(idx)
		return
	}
	// leaf with no children: nothing to do on enter
}

func (a *ApiDocsAdapter) toggleExpand(idx int) {
	if a.items[idx].expanded {
		// collapse: remove all descendants from flat list
		item := a.items[idx]
		item.expanded = false
		end := idx + 1
		for end < len(a.items) && a.items[end].depth > item.depth {
			end++
		}
		newItems := make([]docItem, 0, len(a.items)-(end-idx-1))
		newItems = append(newItems, a.items[:idx]...)
		newItems = append(newItems, item)
		newItems = append(newItems, a.items[end:]...)
		a.items = newItems
	} else {
		// expand: insert children after idx
		item := a.items[idx]
		item.expanded = true
		newItems := make([]docItem, 0, len(a.items)+len(item.children))
		newItems = append(newItems, a.items[:idx]...)
		newItems = append(newItems, item)
		newItems = append(newItems, item.children...)
		newItems = append(newItems, a.items[idx+1:]...)
		a.items = newItems
	}
	a.rebuildSelectIdx()
}

func (a *ApiDocsAdapter) rebuildSelectIdx() {
	a.selectIdx = nil
	for i, item := range a.items {
		if item.kind == docItemParam {
			a.selectIdx = append(a.selectIdx, i)
		}
	}
	if a.cursor >= len(a.selectIdx) && len(a.selectIdx) > 0 {
		a.cursor = len(a.selectIdx) - 1
	}
}


// UpdateApiDocsPort handles scroll messages for the api docs view.
func (a *ApiDocsAdapter) UpdateApiDocsPort(msg tea.Msg) tea.Cmd {
	if !a.ready {
		return nil
	}
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			a.cursorUp()
		case "down", "j":
			a.cursorDown()
		case "pgup":
			a.viewport.HalfViewUp()
		case "pgdn":
			a.viewport.HalfViewDown()
		case "home":
			a.viewport.GotoTop()
			a.cursor = 0
		case "end":
			a.viewport.GotoBottom()
			if len(a.selectIdx) > 0 {
				a.cursor = len(a.selectIdx) - 1
			}
		}
	default:
		var cmd tea.Cmd
		a.viewport, cmd = a.viewport.Update(msg)
		return cmd
	}
	return nil
}

// SetDocSize propagates terminal dimensions to the docs viewport.
func (a *ApiDocsAdapter) SetDocSize(width, height int) {
	a.width = width
	a.height = height
	if a.ready {
		a.viewport.Width = width - 2
		a.viewport.Height = height - 3
	}
}

// ---------------------------------------------------------------------------
// Rendering
// ---------------------------------------------------------------------------

func (a *ApiDocsAdapter) ViewApiDocs(width, height int) string {
	a.width = width
	a.height = height

	innerWidth := width - 2
	innerHeight := height - 3
	if innerWidth < 1 {
		innerWidth = 1
	}
	if innerHeight < 1 {
		innerHeight = 1
	}

	if len(a.items) == 0 {
		return common.BorderizeWithSpinnerCheck("No API documentation available.", true, map[common.BorderPosition]string{
			common.TopMiddleBorder: titleStyle.Render(fmt.Sprintf(" api docs: %s ", a.resourcePath)),
		})
	}

	body := a.renderBody()

	if !a.ready {
		a.viewport = viewport.New(innerWidth, innerHeight)
		a.ready = true
	}
	a.viewport.Width = innerWidth
	a.viewport.Height = innerHeight
	a.viewport.SetContent(body)
	a.scrollToCursor()

	viewContent := a.viewport.View()
	lines := strings.Split(viewContent, "\n")

	opaqueStyle := lipgloss.NewStyle().Width(innerWidth).Background(colors.BlackTerm)
	for len(lines) < innerHeight {
		lines = append(lines, opaqueStyle.Render(strings.Repeat(" ", innerWidth)))
	}
	for i, line := range lines {
		lw := lipgloss.Width(line)
		if lw < innerWidth {
			lines[i] = opaqueStyle.Render(line + strings.Repeat(" ", innerWidth-lw))
		} else if lw > innerWidth {
			lines[i] = opaqueStyle.Render(lipgloss.NewStyle().Width(innerWidth).Render(line))
		} else {
			lines[i] = opaqueStyle.Render(line)
		}
	}

	titleLabel := titleStyle.Render(fmt.Sprintf(" api docs: %s ", a.resourcePath))
	scrollPct := fmt.Sprintf("%.0f%%", a.viewport.ScrollPercent()*100)
	embeddedText := map[common.BorderPosition]string{
		common.TopMiddleBorder:    titleLabel,
		common.BottomRightBorder: scrollPct,
	}

	return common.BorderizeWithSpinnerCheck(strings.Join(lines, "\n"), true, embeddedText)
}

func (a *ApiDocsAdapter) renderBody() string {
	var sb strings.Builder

	for i, item := range a.items {
		switch item.kind {
		case docItemSeparator:
			if item.display != "" {
				sb.WriteString(sectionLabelStyle.Render(item.display))
			}
			sb.WriteByte('\n')

		case docItemSection:
			sb.WriteString(item.display)
			sb.WriteByte('\n')

		case docItemParam:
			// Check if this item is currently selected
			selected := len(a.selectIdx) > 0 && a.selectIdx[a.cursor] == i
			sb.WriteString(a.renderParamLine(item, selected))
			sb.WriteByte('\n')
		}
	}
	return sb.String()
}

func (a *ApiDocsAdapter) renderParamLine(item docItem, selected bool) string {
	indent := strings.Repeat("    ", item.depth)

	prefix := "  "
	if item.isArrayItem {
		prefix = "- "
	}

	req := ""
	if item.required {
		req = requiredStarStyle.Render("*")
	}

	typePart := renderTypeBadge(item.typeName)

	arrow := ""
	if len(item.children) > 0 {
		if item.expanded {
			arrow = expandedArrowStyle.Render(" ▼")
		} else {
			arrow = expandedArrowStyle.Render(" ▶")
		}
	}

	var namePart string
	if selected {
		namePart = selectedNameStyle.Render(item.name)
	} else {
		namePart = fieldNameStyle.Render(item.name)
	}

	desc := ""
	if item.description != "" {
		desc = summaryStyle.Render("  " + item.description)
	}

	// layout: indent  prefix  name[*]  [type][arrow]   description
	return indent + prefix + namePart + req + "  " + typePart + arrow + desc
}

func (a *ApiDocsAdapter) scrollToCursor() {
	if !a.ready || len(a.selectIdx) == 0 {
		return
	}

	// Count the visible line of the selected item
	targetLine := 0
	targetItemIdx := a.selectIdx[a.cursor]
	for i := 0; i < targetItemIdx && i < len(a.items); i++ {
		targetLine++
	}

	// Scroll viewport to keep cursor visible
	vTop := a.viewport.YOffset
	vBot := vTop + a.viewport.Height - 1
	if targetLine < vTop {
		a.viewport.SetYOffset(targetLine)
	} else if targetLine > vBot {
		a.viewport.SetYOffset(targetLine - a.viewport.Height + 1)
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func formatSection(method, path, summary string) string {
	methodStr := methodStyle(method).Render(fmt.Sprintf("[%s]", method))
	pathStr := pathStyle.Render(" " + path)
	sumStr := ""
	if summary != "" {
		sumStr = summaryStyle.Render("  " + summary)
	}
	return methodStr + pathStr + sumStr
}

func paramToItem(p *openapi3.Parameter) docItem {
	typeName := "any"
	var children []docItem
	if p.Schema != nil && p.Schema.Value != nil {
		typeName = schemaTypeName(p.Schema.Value)
		children = buildChildren(p.Schema, 1, p.In)
	}
	desc := p.Description
	if desc == "" && p.Schema != nil && p.Schema.Value != nil {
		desc = p.Schema.Value.Description
	}
	return docItem{
		kind:        docItemParam,
		name:        p.Name,
		typeName:    typeName,
		required:    p.Required,
		description: desc,
		in:          p.In,
		depth:       0,
		children:    children,
	}
}

func schemaToItem(name string, ref *openapi3.SchemaRef, required bool, in string) docItem {
	return schemaToItemAtDepth(name, ref, required, in, 0)
}

func schemaToItemAtDepth(name string, ref *openapi3.SchemaRef, required bool, in string, depth int) docItem {
	typeName := "any"
	desc := ""
	var children []docItem
	if ref != nil && ref.Value != nil {
		typeName = schemaTypeName(ref.Value)
		desc = ref.Value.Description
		if depth < 4 {
			children = buildChildren(ref, depth+1, in)
		}
	}
	return docItem{
		kind:        docItemParam,
		name:        name,
		typeName:    typeName,
		required:    required,
		description: desc,
		in:          in,
		depth:       depth,
		children:    children,
	}
}

func buildChildren(ref *openapi3.SchemaRef, depth int, in string) []docItem {
	if ref == nil || ref.Value == nil {
		return nil
	}
	s := ref.Value
	var objSchema *openapi3.Schema
	isArray := false
	if s.Type != nil {
		for _, t := range *s.Type {
			switch t {
			case "object":
				objSchema = s
			case "array":
				if s.Items != nil && s.Items.Value != nil && isObjectSchema(s.Items.Value) {
					objSchema = s.Items.Value
					isArray = true
				}
			}
		}
	}
	if objSchema == nil || len(objSchema.Properties) == 0 {
		return nil
	}
	reqSet := map[string]bool{}
	for _, r := range objSchema.Required {
		reqSet[r] = true
	}
	type prop struct {
		name string
		ref  *openapi3.SchemaRef
	}
	var props []prop
	for n, r := range objSchema.Properties {
		props = append(props, prop{n, r})
	}
	sort.Slice(props, func(i, j int) bool { return props[i].name < props[j].name })
	var children []docItem
	for i, p := range props {
		item := schemaToItemAtDepth(p.name, p.ref, reqSet[p.name], in, depth)
		item.isArrayItem = isArray && i == 0 // dash only on first item
		children = append(children, item)
	}
	return children
}

func isObjectSchema(s *openapi3.Schema) bool {
	if s == nil {
		return false
	}
	if s.Type == nil {
		return len(s.Properties) > 0
	}
	for _, t := range *s.Type {
		if t == "object" {
			return true
		}
	}
	return false
}

func renderTypeBadge(typeName string) string {
	abbr := abbreviateType(typeName)
	open := strings.Index(abbr, "(")
	if open == -1 {
		return typeStyle.Render("[" + abbr + "]")
	}
	outer := abbr[:open]
	inner := abbr[open:]
	return typeStyle.Render("["+outer) + typeInnerStyle.Render(inner) + typeStyle.Render("]")
}

func abbreviateType(t string) string {
	replacer := strings.NewReplacer(
		"string", "str",
		"integer", "int",
		"boolean", "bool",
		"number", "num",
		"array", "arr",
		"object", "obj",
	)
	return replacer.Replace(t)
}

func schemaTypeName(s *openapi3.Schema) string {
	if s == nil {
		return "any"
	}
	if s.Type != nil {
		for _, t := range *s.Type {
			if t == "" {
				continue
			}
			if t == "array" && s.Items != nil && s.Items.Value != nil {
				inner := schemaTypeName(s.Items.Value)
				return "array(" + inner + ")"
			}
			if s.Format != "" {
				return t + "(" + s.Format + ")"
			}
			return t
		}
	}
	return "any"
}


// ---------------------------------------------------------------------------
// Styles
// ---------------------------------------------------------------------------

var (
	titleStyle        = lipgloss.NewStyle().Background(colors.Orange).Foreground(colors.BlackTerm).Bold(true)
	sectionLabelStyle = lipgloss.NewStyle().Foreground(colors.LightGrey)
	pathStyle    = lipgloss.NewStyle().Foreground(colors.VeryLightGrey)
	summaryStyle = lipgloss.NewStyle().Foreground(colors.Grey240).Italic(true)
	typeStyle      = lipgloss.NewStyle().Foreground(colors.MediumCyan)
	typeInnerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#f0a500"))
	requiredStarStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#f93e3e"))
	fieldNameStyle      = lipgloss.NewStyle().Foreground(colors.VeryLightGrey)
	selectedNameStyle   = lipgloss.NewStyle().Foreground(colors.VeryLightGrey).Background(colors.DarkGreenBlue)
	expandedArrowStyle = lipgloss.NewStyle().Foreground(colors.MediumCyan)
)

func methodStyle(method string) lipgloss.Style {
	switch method {
	case "GET":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#61affe")).Bold(true)
	case "POST":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#49cc90")).Bold(true)
	case "PUT":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#fca130")).Bold(true)
	case "PATCH":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#50e3c2")).Bold(true)
	case "DELETE":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#f93e3e")).Bold(true)
	case "HEAD":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#9012fe")).Bold(true)
	case "OPTIONS":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#0d5aa7")).Bold(true)
	default:
		return lipgloss.NewStyle().Foreground(colors.LightGrey).Bold(true)
	}
}
