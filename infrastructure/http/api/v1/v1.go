package v1

import (
	"github.com/Testzyler/order-management-go/application/constants"
	"github.com/gofiber/fiber/v2"
)

type Routes []Route

type Route struct {
	Name        string
	Path        string
	Method      string
	HandlerFunc constants.HandlerFunc
}

type RouteDefinition struct {
	Routes Routes
	Prefix string
}

var RouteDefinitions = make([]RouteDefinition, 0)

func AddRoute(router *fiber.Router) {
	for _, routeDefinition := range RouteDefinitions {
		routerWithPrefix := (*router).Group(routeDefinition.Prefix)
		for _, route := range routeDefinition.Routes {
			if route.Method == constants.METHOD_GET {
				routerWithPrefix.Get(route.Path, route.HandlerFunc)
			} else if route.Method == constants.METHOD_POST {
				routerWithPrefix.Post(route.Path, route.HandlerFunc)
			} else if route.Method == constants.METHOD_DELETE {
				routerWithPrefix.Delete(route.Path, route.HandlerFunc)
			} else if route.Method == constants.METHOD_PUT {
				routerWithPrefix.Put(route.Path, route.HandlerFunc)
			}
		}
	}
}
