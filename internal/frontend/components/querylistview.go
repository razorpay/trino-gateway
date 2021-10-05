package components

import (
	"fmt"

	"github.com/hexops/vecty"
	"github.com/hexops/vecty/elem"
	"github.com/hexops/vecty/prop"
	"github.com/razorpay/trino-gateway/internal/frontend/core"
)

// QueryListView is a vecty.Component which represents the query history section
type QueryListView struct {
	vecty.Core
	core  core.ICore
	items vecty.List
}

func (p *QueryListView) Render() vecty.ComponentOrHTML {
	queries, err := p.core.GetAllQueries()
	if err != nil {
		fmt.Println(err.Error())
		return vecty.Text(fmt.Sprintf("%s: %s", "Unable to fetch list of queries.", err.Error()))
	}
	for _, q := range queries {
		query := &QueryView{
			Query: q,
		}
		p.items = append(p.items, query)
	}

	return elem.Div(
		vecty.Markup(
			vecty.Class("container", "tile", "is-vertical", "is-ancestor"),
		),
		p.renderHeader(),
		p.renderItems(),
		p.renderPagination(),
	)
}

func (p *QueryListView) renderHeader() vecty.ComponentOrHTML {
	return elem.Div(
		vecty.Markup(
			vecty.Class("tile", "is-parent"),
		),
		elem.Div(
			vecty.Markup(
				vecty.Class("tile", "is-child"),
			),
			vecty.Text(fmt.Sprintf("Total: %d", len(p.items))),
		),
		elem.Div(
			vecty.Markup(
				vecty.Class("tile", "is-child"),
			),
			elem.Input(
				vecty.Markup(
					vecty.Class("input"),
					prop.Type(prop.TypeText),
					prop.Placeholder("Search for Username"), // initial textarea text.
				),

				// When input is typed into the textarea, update the local
				// component state and rerender.
				// event.Input(func(e *vecty.Event) {
				// 	p.Input = e.Target.Get("value").String()
				// 	vecty.Rerender(p)
				// }),
			),
		),
		elem.Div(
			vecty.Markup(
				vecty.Class("tile", "is-child"),
			),
			elem.Div(
				vecty.Markup(
					vecty.Class("select", "is-rounded"),
				),
				elem.Select(
					elem.Option(vecty.Text("10"+" Entries per page")),
					elem.Option(vecty.Text("50"+" Entries per page")),
					elem.Option(vecty.Text("100"+" Entries per page")),
				),
			),

			// 			<div class="select is-multiple">
			//   <select multiple size="8">
			//     <option value="Argentina">Argentina</option>
			//     <option value="Bolivia">Bolivia</option>
			//     <option value="Brazil">Brazil</option>
			//     <option value="Chile">Chile</option>
			//     <option value="Colombia">Colombia</option>
			//     <option value="Ecuador">Ecuador</option>
			//     <option value="Guyana">Guyana</option>
			//     <option value="Paraguay">Paraguay</option>
			//     <option value="Peru">Peru</option>
			//     <option value="Suriname">Suriname</option>
			//     <option value="Uruguay">Uruguay</option>
			//     <option value="Venezuela">Venezuela</option>
			//   </select>
			// </div>
		),
	)
}

func (p *QueryListView) onEditSearchbox(e vecty.Event) {
}

func (p *QueryListView) renderItems() vecty.ComponentOrHTML {
	return elem.OrderedList(p.items)
}

func (p *QueryListView) renderPagination() vecty.ComponentOrHTML {
	totPag := fmt.Sprint(len(p.items))
	currPag := fmt.Sprint(4)

	return elem.Div(
		vecty.Markup(
			vecty.Class("tile", "is-parent"),
		),
		elem.Div(
			vecty.Markup(
				vecty.Class("tile", "is-child"),
			),
			elem.Navigation(
				vecty.Markup(
					vecty.Class("pagination", "is-rounded"),
					vecty.Property("role", "navigation"),
					vecty.Property("aria-label", "pagination"),
				),
				elem.Anchor(
					vecty.Markup(vecty.Class("pagination-previous")),
					vecty.Text("Previous"),
				),
				elem.Anchor(
					vecty.Markup(vecty.Class("pagination-next")),
					vecty.Text("Next page"),
				),
				elem.UnorderedList(
					vecty.Markup(
						vecty.Class("pagination-list"),
					),
					elem.ListItem(elem.Anchor(
						vecty.Markup(
							vecty.Class("pagination-link"),
							vecty.Property("aria-label", "Goto page 1"),
						),
						vecty.Text("1"),
					)),
					elem.ListItem(elem.Span(
						vecty.Markup(
							vecty.Class("pagination-ellipsis"),
						),
						vecty.Text("..."),
					)),
					elem.ListItem(elem.Anchor(
						vecty.Markup(
							vecty.Class("pagination-link", "is-current"),
							vecty.Property("aria-label", "Goto page "+currPag),
						),
						vecty.Text(currPag),
					)),
					elem.ListItem(elem.Span(
						vecty.Markup(
							vecty.Class("pagination-ellipsis"),
						),
						vecty.Text("..."),
					)),
					elem.ListItem(elem.Anchor(
						vecty.Markup(
							vecty.Class("pagination-link"),
							vecty.Property("aria-label", "Goto page "+totPag),
						),
						vecty.Text(totPag),
					)),
				),
			),
		),
	)
}
