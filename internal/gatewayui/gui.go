package gatewayui

import (
	"html/template"
	"net/http"

	"github.com/razorpay/trino-gateway/internal/gatewayserver/models"
)

type todoPageData struct {
	PageTitle string
	Queries   []models.Query
}

func HttpHandler(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles("web/gatewayui/layout.html"))

	data := todoPageData{
		PageTitle: "Trino Gateway",
		Queries: []models.Query{
			{Text: "Task 1"},
			{Text: "Task 2"},
			{Text: "Task 3"},
		},
	}
	tmpl.Execute(w, data)
}

// func listAllQueries() []models.Query {

// }
