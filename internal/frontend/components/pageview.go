package components

import (
	"github.com/hexops/vecty"
	"github.com/hexops/vecty/elem"
	"github.com/razorpay/trino-gateway/internal/frontend/core"
)

// PageView is a vecty.Component which represents the entire page.
type PageView struct {
	vecty.Core
	core core.ICore
}

func GetNewPageViewComponent(c core.ICore) *PageView {
	return &PageView{core: c}
}

func (p *PageView) Render() vecty.ComponentOrHTML {
	return elem.Body(
		elem.Section(
			vecty.Markup(
				vecty.Class("section"),
			),
			elem.Div(
				vecty.Markup(
					vecty.Class("container"),
				),
				p.renderHeader(),
				NewQueryListView(p.core),
				p.renderFooter(),
			),
		),
	)
}

func (p *PageView) renderHeader() *vecty.HTML {
	return elem.Div(
		vecty.Markup(
			vecty.Class("tabs", "is-centered", "is-large", "is-fullwidth", "is-toggle", "is-toggle-rounded"),
		),
		elem.UnorderedList(
			elem.ListItem(
				vecty.Markup(
					vecty.Class("is-active"),
				),
				&TabView{title: "Query History", hrefUrl: "#"},
			),
			elem.ListItem(&TabView{title: "Dashboard", hrefUrl: "#"}),
			elem.ListItem(&TabView{title: "Admin", hrefUrl: "/admin/swaggerui"}),
		),
	)
}

func (p *PageView) renderFooter() *vecty.HTML {
	return elem.Footer(
		vecty.Markup(
			vecty.Class("footer"),
		),
		elem.Div(
			vecty.Markup(
				vecty.Class("content", "has-text-centered"),
			),
			vecty.Text("trino gateway footer"),
		),
	)
}
