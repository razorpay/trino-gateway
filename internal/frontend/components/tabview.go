package components

import (
	"github.com/hexops/vecty"
	"github.com/hexops/vecty/elem"
)

// TabView is a vecty.Component which represents a single elements in the tabBar
type TabView struct {
	vecty.Core
	title string
}

func (p *TabView) Render() vecty.ComponentOrHTML {
	return elem.Anchor(
		vecty.Text(p.title),
	)
}
