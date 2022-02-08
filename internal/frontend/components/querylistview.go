package components

import (
	"fmt"
	"math"

	"github.com/hexops/vecty"
	"github.com/hexops/vecty/elem"
	"github.com/hexops/vecty/event"
	"github.com/hexops/vecty/prop"
	"github.com/razorpay/trino-gateway/internal/frontend/core"
)

// QueryListView is a vecty.Component which represents the query history section
type queryListView struct {
	vecty.Core
	core   core.ICore
	items  vecty.List
	params queryListViewParams
}

type queryListViewParams struct {
	Username     string
	MaxItems     int
	TotalItems   int
	PageIndex    int
	ItemsPerPage int
}

func NewQueryListView(core core.ICore) *queryListView {
	p := &queryListView{
		core: core,
		params: queryListViewParams{
			PageIndex:    0,
			ItemsPerPage: 100,
			MaxItems:     1000,
		},
	}

	if err := p.populateItems(); err != nil {
		fmt.Printf("%s: %s\n", "Unable to fetch list of queries.", err.Error())
	}
	return p
}

func (p *queryListView) populateItems() error {
	// Get total eligible items
	queries, err := p.core.GetQueries(p.params.MaxItems, 0, p.params.Username)
	if err != nil {
		return err
	}
	p.params.TotalItems = len(queries)
	// queries, err = p.core.GetQueries(p.params.ItemsPerPage, p.params.PageIndex*p.params.ItemsPerPage, p.params.Username)
	// if err != nil {
	// 	return err
	// }
	for _, q := range queries {
		query := &QueryView{
			Query: q,
		}
		p.items = append(p.items, query)
	}
	return nil
}

func (p *queryListView) Render() vecty.ComponentOrHTML {
	return elem.Div(
		vecty.Markup(
			vecty.Class("container", "tile", "is-vertical", "is-ancestor"),
		),
		p.renderHeader(),
		p.renderItems(),
		p.renderPagination(),
	)
}

func (p *queryListView) renderHeader() vecty.ComponentOrHTML {
	return elem.Div(
		vecty.Markup(
			vecty.Class("tile", "is-parent"),
		),
		elem.Div(
			vecty.Markup(
				vecty.Class("tile", "is-child"),
			),
			vecty.Text(fmt.Sprintf("Total: %d", p.params.TotalItems)),
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
					elem.Option(vecty.Text("100"+" Entries per page")),
					elem.Option(vecty.Text("200"+" Entries per page")),
					elem.Option(vecty.Text("500"+" Entries per page")),
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

func (p *queryListView) onEditSearchbox(e *vecty.Event) {
}

func (p *queryListView) onClickPageNavigation(_ *vecty.Event) {
	p.params.PageIndex = p.params.PageIndex + 1
	vecty.Rerender(p)
}

func (p *queryListView) renderItems() vecty.ComponentOrHTML {
	r := vecty.List{}
	for i, v := range p.items {
		if i >= p.params.PageIndex*p.params.ItemsPerPage && i < (p.params.PageIndex+1)*p.params.ItemsPerPage {
			r = append(r, v)
		}
	}
	return elem.OrderedList(r)
}

func (p *queryListView) renderPagination() vecty.ComponentOrHTML {
	totPag := int(math.Ceil(float64(p.params.TotalItems) / float64(p.params.ItemsPerPage)))
	currPag := p.params.PageIndex + 1

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
					vecty.Markup(
						vecty.MarkupIf(currPag == 1, vecty.Style("display", "none")),
						vecty.Class("pagination-previous"),
					),
					vecty.Text("Previous"),
				),
				elem.Anchor(
					vecty.Markup(
						vecty.MarkupIf(currPag == totPag, vecty.Style("display", "none")),
						vecty.Class("pagination-next"),
						event.Click(p.onClickPageNavigation).PreventDefault(),
					),
					vecty.Text("Next page"),
				),
				elem.UnorderedList(
					vecty.Markup(
						vecty.Class("pagination-list"),
					),
					elem.ListItem(elem.Anchor(
						vecty.Markup(
							vecty.MarkupIf(currPag == 1, vecty.Style("display", "none")),
							vecty.Class("pagination-link"),
							vecty.Property("aria-label", "Goto page 1"),
						),
						vecty.Text("1"),
					)),
					elem.ListItem(elem.Span(
						vecty.Markup(
							vecty.MarkupIf(currPag == 1, vecty.Style("display", "none")),
							vecty.Class("pagination-ellipsis"),
						),
						vecty.Text("..."),
					)),
					elem.ListItem(elem.Anchor(
						vecty.Markup(
							vecty.Class("pagination-link", "is-current"),
							vecty.Property("aria-label", fmt.Sprint("Goto page ", currPag)),
						),
						vecty.Text(fmt.Sprint(currPag)),
					)),
					elem.ListItem(elem.Span(
						vecty.Markup(
							vecty.MarkupIf(currPag == totPag, vecty.Style("display", "none")),
							vecty.Class("pagination-ellipsis"),
						),
						vecty.Text("..."),
					)),
					elem.ListItem(elem.Anchor(
						vecty.Markup(
							vecty.MarkupIf(currPag == totPag, vecty.Style("display", "none")),
							vecty.Class("pagination-link"),
							vecty.Property("aria-label", fmt.Sprint("Goto page ", totPag)),
						),
						vecty.Text(fmt.Sprint(totPag)),
					)),
				),
			),
		),
	)
}
