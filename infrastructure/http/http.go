package http

import (
	"fmt"
	"os"

	"github.com/Testzyler/order-management-go/application/constants"
	"github.com/Testzyler/order-management-go/infrastructure/http/api"
	"github.com/gofiber/fiber/v2"
	"github.com/spf13/viper"
)

var AppServer *fiber.App
var methodRoutes map[string]map[string]constants.HandlerFunc

func InitHttpServer() {

	// Config Port and Address
	httpPort := viper.GetString("HttpServer.Port")
	AppServer = fiber.New(fiber.Config{
		ReadBufferSize:  1024 * 1024, // 1MB
		WriteBufferSize: 1024 * 1024, // 1MB
	})

	// Config Default Path
	AddRoute()

	// Add Api Path
	apiGroup := AppServer.Group("/api")
	api.AddRoute(&apiGroup)

	// Start Server
	fmt.Printf("serving http at http://127.0.0.1:%s", httpPort)
	err := AppServer.Listen(":" + httpPort)
	if err != nil {
		fmt.Println("http server listen and serves failed")
		os.Exit(1)
	}
}

func ShutdownHttpServer() {
	fmt.Println("http server is shutting down")
	if err := AppServer.Shutdown(); err != nil {
		fmt.Printf("http server shut down failed: %s", err)
		return
	}
	fmt.Println("http server shut down completed")
}

func AddRoute() {
	for method, routes := range methodRoutes {
		if method == constants.METHOD_GET {
			for routeName, routeFunc := range routes {
				AppServer.Get(routeName, routeFunc)
			}
		} else if method == constants.METHOD_POST {
			for routeName, routeFunc := range routes {
				AppServer.Post(routeName, routeFunc)
			}
		}
	}
}

func init() {
	methodRoutes = make(map[string]map[string]constants.HandlerFunc)
	methodRoutes[constants.METHOD_GET] = make(map[string]constants.HandlerFunc)
	methodRoutes[constants.METHOD_POST] = make(map[string]constants.HandlerFunc)

	methodRoutes[constants.METHOD_GET]["/healthz"] = Healthz
}
