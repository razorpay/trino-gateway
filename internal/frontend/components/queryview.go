package components

import (
	"fmt"
	"time"

	"github.com/hexops/vecty"
	"github.com/hexops/vecty/elem"
	"github.com/hexops/vecty/event"
	"github.com/hexops/vecty/prop"
	"github.com/hexops/vecty/style"
	gatewayv1 "github.com/razorpay/trino-gateway/rpc/gateway"
)

// QueryView is a vecty.Component which represents a single item in the queryHistory List.
type QueryView struct {
	vecty.Core
	// core core.ICore

	Query   *gatewayv1.Query
	classes vecty.ClassMap
}

/*
QueryId - href to trinoUi
User
backendId
GroupId
submittedAt
Text
*/

func (p *QueryView) Render() vecty.ComponentOrHTML {
	return elem.Div(
		vecty.Markup(
			vecty.Class("box", "tile", "is-parent", "notification", "is-light"),
			vecty.Style("display", "flex"),
			vecty.Style("flex-direction", "row"),
		),
		p.renderMeta(),
		p.renderText(),
	)
}

type QueryMetaItem struct {
	vecty.Core
	k string
	v string
}

// TODO : FIX it
var classes = vecty.ClassMap{
	"is-info":  true,
	"is-light": true,
	"is-link":  false,
}

func (q *QueryMetaItem) Render() vecty.ComponentOrHTML {
	return vecty.Text(q.v)
}

func (p *QueryView) renderMeta() *vecty.HTML {
	// https://trino-gateway.de.razorpay.com/ui/query.html?20211003_083931_06657_n3mb3
	url := fmt.Sprintf("%s/ui/query.html?%s", p.Query.ServerHost, p.Query.Id)

	// fails at runtime in js
	// loc, _ := time.LoadLocation("Asia/Kolkata")

	_items := [...]*QueryMetaItem{
		{
			k: "SubmittedAt",
			v: time.Unix(p.Query.GetSubmittedAt(), 0).Local().Format("2006/01/02 15:04:05"), // Golang time format layout is weird stuff.
		},
		{k: "Username", v: p.Query.GetUsername()},
		{k: "BackendId", v: p.Query.GetBackendId()},
		{k: "GroupId", v: p.Query.GetGroupId()},
	}

	var items vecty.List
	for _, i := range _items {
		item := elem.ListItem(i)
		items = append(items, item)
	}

	p.classes = classes

	return elem.Div(
		vecty.Markup(
			p.classes,
			vecty.Class("tile", "is-child", "notification"),
			style.Width("30%"),
			vecty.Style("display", "flex"),
			vecty.Style("flex-direction", "column"),
			event.PointerEnter(p.onPointerEnter),
			event.PointerLeave(p.onPointerLeave),
		),
		elem.Bold(
			vecty.Markup(
				vecty.Class("subtitle"),
			),
			elem.Anchor(
				vecty.Markup(
					prop.Href(url),
				),
				vecty.Text(p.Query.GetId()),
			),
		),
		elem.UnorderedList(items),
	)
}

func (p *QueryView) renderText() *vecty.HTML {
	return elem.Div(
		vecty.Markup(
			vecty.Class("tile", "is-child", "is-8"),
			style.Width("70%"),
			style.Height("6.5em"),
			vecty.Style("word-wrap", "break-word"),
			style.Overflow(style.OverflowHidden),
			vecty.Style("text-overflow", "ellipsis"),
			vecty.Style("resize", "vertical"),
			vecty.Style("text-align", "center"),
		),
		vecty.Text(p.Query.GetText()),
	)
}

func (p *QueryView) onPointerEnter(e *vecty.Event) {
	p.classes["is-info"] = false
	p.classes["is-link"] = true
	vecty.Rerender(p)
}

func (p *QueryView) onPointerLeave(e *vecty.Event) {
	p.classes["is-info"] = true
	p.classes["is-link"] = false
	vecty.Rerender(p)
}
