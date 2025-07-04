package route

import (
	"github.com/Testzyler/order-management-go/application/constants"
	"github.com/Testzyler/order-management-go/infrastructure/database"
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

// HandlerInitializer interface that all handlers must implement
type HandlerInitializer interface {
	Initialize()
	GetRouteDefinition() RouteDefinition
}

// HandlerRegistry holds all registered handlers
type HandlerRegistry struct {
	handlers []HandlerInitializer
}

var registry = &HandlerRegistry{
	handlers: make([]HandlerInitializer, 0),
}

// RegisterHandler adds a handler to the registry
func RegisterHandler(handler HandlerInitializer) {
	registry.handlers = append(registry.handlers, handler)
}

// InitializeAllHandlers initializes all registered handlers
// This should be called after the database connection is established
func InitializeAllHandlers() {
	// Clear existing route definitions
	RouteDefinitions = make([]RouteDefinition, 0)

	// Initialize all registered handlers
	for _, handler := range registry.handlers {
		handler.Initialize()
		routeDefinition := handler.GetRouteDefinition()
		RouteDefinitions = append(RouteDefinitions, routeDefinition)
	}
}

func GetDatabasePool() database.DatabaseInterface {
	return database.DatabasePool
}

func AddRoutesPrefix(router *fiber.Router) fiber.Router {
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
	return *router
}
