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
		Todos: []todo{
			{Title: "Task 1", Done: false},
			{Title: "Task 2", Done: true},
			{Title: "Task 3", Done: true},
		},
	}
	tmpl.Execute(w, data)
}

func listAllQueries() []models.Query {

}
