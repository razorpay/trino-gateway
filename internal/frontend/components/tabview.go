package components

import (
	"github.com/hexops/vecty"
	"github.com/hexops/vecty/elem"
	"github.com/hexops/vecty/event"
	"github.com/hexops/vecty/prop"
)

// TabView is a vecty.Component which represents a single elements in the tabBar
type TabView struct {
	vecty.Core
	title      string
	isSelected bool
	component  *vecty.ComponentOrHTML
}

func (p *TabView) Render() vecty.ComponentOrHTML {
	return elem.Anchor(
		vecty.Markup(
			vecty.MarkupIf(p.isSelected, vecty.Class("is-active")),
			prop.Href("#"),
			event.Click(p.onClick).PreventDefault(),
		),
		vecty.Text(p.title),
	)
}

func (p *TabView) onClick(e *vecty.Event) {
}
