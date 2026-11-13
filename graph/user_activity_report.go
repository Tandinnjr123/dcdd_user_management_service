package schema

import "github.com/graphql-go/graphql"

var DcddUserActivityReportRow = graphql.NewObject(graphql.ObjectConfig{
	Name: "DcddUserActivityReportRow",
	Fields: graphql.Fields{
		"user_id":       &graphql.Field{Type: graphql.String},
		"name":          &graphql.Field{Type: graphql.String},
		"cid":           &graphql.Field{Type: graphql.String},
		"email":         &graphql.Field{Type: graphql.String},
		"mobile_no":     &graphql.Field{Type: graphql.String},
		"category":      &graphql.Field{Type: graphql.String},
		"status":        &graphql.Field{Type: graphql.String},
		"last_login_at": &graphql.Field{Type: graphql.DateTime},
	},
})

var DcddUserActivityPagination = graphql.NewObject(graphql.ObjectConfig{
	Name: "DcddUserActivityPagination",
	Fields: graphql.Fields{
		"current_page": &graphql.Field{Type: graphql.Int},
		"total_page":   &graphql.Field{Type: graphql.Int},
		"limit":        &graphql.Field{Type: graphql.Int},
	},
})

var DcddUserActivitiesResult = graphql.NewObject(graphql.ObjectConfig{
	Name: "DcddUserActivitiesResult",
	Fields: graphql.Fields{
		"users":      &graphql.Field{Type: graphql.NewList(DcddUserActivityReportRow)},
		"total":      &graphql.Field{Type: graphql.Int},
		"pagination": &graphql.Field{Type: DcddUserActivityPagination},
	},
})

var DcddUserActivitiesResponse = graphql.NewObject(graphql.ObjectConfig{
	Name: "DcddUserActivitiesResponse",
	Fields: graphql.Fields{
		"data":  &graphql.Field{Type: DcddUserActivitiesResult},
		"error": &graphql.Field{Type: DcddAuthError},
	},
})
