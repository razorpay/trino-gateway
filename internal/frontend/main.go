// gopherjs doesnt build with anythign other than package `main`
package main

import (
	"github.com/gopherjs/gopherjs/js"
	"github.com/hexops/vecty"
	"github.com/razorpay/trino-gateway/internal/frontend/components"
	"github.com/razorpay/trino-gateway/internal/frontend/core"
)

func main() {
	path := accessURL() // fmt.Sprint("http://localhost:", "28000")
	c := core.NewCore(path)

	vecty.SetTitle("Trino-Gateway")
	// vecty.AddStylesheet("https://rawgit.com/tastejs/todomvc-common/master/base.css")
	// vecty.AddStylesheet("https://rawgit.com/tastejs/todomvc-app-css/master/index.css")
	vecty.AddStylesheet("https://cdn.jsdelivr.net/npm/bulma@0.9.3/css/bulma.min.css")

	vecty.RenderBody(components.GetNewPageViewComponent(c))
}

var location = js.Global.Get("location")

func accessURL() string {
	// current URL: http://localhost:8000/code/gopherjs/window-location/index.html?a=1

	// return - http://localhost:8000/code/gopherjs/window-location/index.html?a=1
	location.Get("href").String()
	// return - localhost:8000
	location.Get("host").String()
	// return - localhost
	location.Get("hostname").String()
	// return - /code/gopherjs/window-location/index.html
	location.Get("pathname").String()
	// return - http:
	location.Get("protocol").String()
	// return - http://localhost:8000
	location.Get("origin").String()
	// return - 8000
	location.Get("port").String()
	// return - ?a=1
	location.Get("search").String()

	return location.Get("origin").String()
}
