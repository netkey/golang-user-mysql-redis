package handler

import (
	"github.com/graphql-go/graphql"
	"your_project/internal/service"
)

func NewSchema(svc *service.UserService) (graphql.Schema, error) {
	userType := graphql.NewObject(graphql.ObjectConfig{
		Name: "User",
		Fields: graphql.Fields{
			"id":    &graphql.Field{Type: graphql.Int},
			"name":  &graphql.Field{Type: graphql.String},
			"email": &graphql.Field{Type: graphql.String},
		},
	})

	return graphql.NewSchema(graphql.SchemaConfig{
		Query: graphql.NewObject(graphql.ObjectConfig{
			Name: "Query",
			Fields: graphql.Fields{
				"user": &graphql.Field{
					Type: userType,
					Args: graphql.FieldConfigArgument{"id": &graphql.ArgumentConfig{Type: graphql.Int}},
					Resolve: func(p graphql.ResolveParams) (interface{}, error) {
						return svc.GetUser(p.Context, p.Args["id"].(int))
					},
				},
			},
		}),
		Mutation: graphql.NewObject(graphql.ObjectConfig{
			Name: "Mutation",
			Fields: graphql.Fields{
				"updateUser": &graphql.Field{
					Type: userType,
					Args: graphql.FieldConfigArgument{
						"id":   &graphql.ArgumentConfig{Type: graphql.Int},
						"name": &graphql.ArgumentConfig{Type: graphql.String},
					},
					Resolve: func(p graphql.ResolveParams) (interface{}, error) {
						return svc.UpdateUser(p.Context, p.Args["id"].(int), p.Args["name"].(string))
					},
				},
			},
		}),
	})
}
