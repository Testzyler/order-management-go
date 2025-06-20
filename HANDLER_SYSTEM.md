# Automatic Handler Registration System

This system allows you to add new API handlers without manually registering them each time. The database connection is established before any handlers are initialized, solving the dependency injection problem.

## How It Works

### 1. Handler Registry (`infrastructure/http/api/v1/registry.go`)

The registry system provides:
- `HandlerInitializer` interface that all handlers must implement
- `RegisterHandler()` function to auto-register handlers
- `InitializeAllHandlers()` function called after database is ready
- `GetDatabasePool()` helper function to access the database

### 2. Handler Interface

Every handler must implement:

```go
type HandlerInitializer interface {
    Initialize()                      // Called after database is ready
    GetRouteDefinition() RouteDefinition // Returns routes for this handler
}
```

### 3. Application Startup Sequence

The startup sequence in `cmd/http_serve.go`:

1. **Initialize Database** - `initPostgresql()`
2. **Initialize HTTP Server** - `initHttpServer()`
3. **Initialize All Handlers** - `v1.InitializeAllHandlers()`

This ensures the database connection is available before any handler tries to use it.

## Adding a New Handler

To add a new handler (e.g., ProductHandler):

### 1. Create the Handler File

Create `infrastructure/http/api/v1/product.go`:

```go
package v1

import (
    "github.com/Testzyler/order-management-go/application/constants"
    "github.com/gofiber/fiber/v2"
)

type ProductHandler struct {
    // service domain.ProductService
}

func NewProductHandler() *ProductHandler {
    return &ProductHandler{}
}

// Initialize implements HandlerInitializer interface
func (h *ProductHandler) Initialize() {
    // Initialize your dependencies here after database is ready
    // repo := repositories.NewProductRepository(GetDatabasePool())
    // service := services.NewProductService(repo)
    // h.service = service
}

// GetRouteDefinition implements HandlerInitializer interface
func (h *ProductHandler) GetRouteDefinition() RouteDefinition {
    return RouteDefinition{
        Routes: Routes{
            Route{
                Name:        "CreateProduct",
                Path:        "/",
                Method:      constants.METHOD_POST,
                HandlerFunc: h.CreateProduct,
            },
            Route{
                Name:        "GetProduct",
                Path:        "/:id",
                Method:      constants.METHOD_GET,
                HandlerFunc: h.GetProduct,
            },
        },
        Prefix: "products",
    }
}

func (h *ProductHandler) CreateProduct(c *fiber.Ctx) error {
    // Your logic here
    return c.JSON(fiber.Map{"message": "Product created"})
}

func (h *ProductHandler) GetProduct(c *fiber.Ctx) error {
    // Your logic here
    return c.JSON(fiber.Map{"message": "Get product"})
}

// Auto-register the handler - this is all you need!
func init() {
    RegisterHandler(NewProductHandler())
}
```

### 2. That's It!

No manual registration needed! The handler will be automatically:
- Registered during package initialization
- Initialized after database connection is ready
- Routes added to the router

## Benefits

1. **No Manual Registration**: Just add the `init()` function with `RegisterHandler()`
2. **Database Always Ready**: Handlers are initialized after database connection is established
3. **Clean Separation**: Each handler is self-contained
4. **Scalable**: Easy to add new features without touching existing code
5. **Type Safe**: Interface ensures all handlers follow the same pattern

## Running the Application

Use the new serve command:

```bash
# Build
go build -o bin/order-cli .

# Run with serve command
./bin/order-cli http-serve

# Or use the alternative serve command
./bin/order-cli serve
```

The application will:
1. Initialize database connection
2. Automatically discover and initialize all handlers
3. Start the HTTP server
4. Handle graceful shutdown

## API Endpoints

With the current handlers, you'll have:

- **Orders**: `/api/v1/orders/`
- **Users**: `/api/v1/users/` (example)
- **Products**: `/api/v1/products/` (when you add it)

All handlers are automatically registered and initialized!
