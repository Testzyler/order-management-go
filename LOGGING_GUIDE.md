# Structured Logging Implementation Guide

This document explains how to use the structured logging system implemented with `slog` in your Go application.

## Overview

The logging system provides:
- Structured logging with JSON and text formats
- Request ID tracking across the entire request lifecycle
- Contextual logging with custom fields
- Component-based logging
- Automatic HTTP request logging
- Error tracking and debugging capabilities

## Configuration

The logger is configured in `config/config.yaml`:

```yaml
Logger:
  Level: debug          # debug, info, warn, error
  Format: json         # json or text
  AddSource: true      # adds source file and line number
  TimeFormat: "2006-01-02T15:04:05.000Z"
  Output: stdout       # stdout, stderr, or file path
```

## Usage Examples

### 1. Basic Logging

```go
import "github.com/Testzyler/order-management-go/infrastructure/logger"

// Simple logging
logger.Info("Application started")
logger.Debug("Debug information")
logger.Warn("Warning message")
logger.Error("Error occurred")

// With additional fields
logger.Info("User logged in", "user_id", "12345", "ip", "192.168.1.1")
```

### 2. Component-Based Logging

```go
// Create a logger for a specific component
dbLogger := logger.WithComponent("database")
dbLogger.Info("Database connection established")

serviceLogger := logger.WithComponent("order-service")
serviceLogger.Info("Processing order", "order_id", 123)
```

### 3. Contextual Logging with Fields

```go
// Single field
userLogger := logger.WithField("user_id", "12345")
userLogger.Info("User action performed")

// Multiple fields
requestLogger := logger.WithFields(map[string]interface{}{
    "request_id": "req-123",
    "user_id": "user-456",
    "action": "create_order",
})
requestLogger.Info("Request processed")
```

### 4. HTTP Handler Usage (Fiber)

```go
func (h *OrderHandler) CreateOrder(c *fiber.Ctx) error {
    // Get logger from request context (includes request ID automatically)
    logger := middleware.GetLoggerFromFiberContext(c).WithComponent("order-handler")
    
    logger.Info("Creating new order")
    
    var input models.CreateOrderInput
    if err := c.BodyParser(&input); err != nil {
        logger.WithError(err).Error("Failed to parse request body")
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
            "message": err.Error(),
        })
    }
    
    logger.WithFields(map[string]interface{}{
        "customer_name": input.CustomerName,
        "item_count": len(input.Items),
    }).Debug("Parsed order input")
    
    // ... rest of handler logic
    
    logger.Info("Order created successfully")
    return c.Status(fiber.StatusCreated).JSON(fiber.Map{
        "message": "Order created successfully",
    })
}
```

### 5. Service Layer Usage

```go
func (s *OrderService) CreateOrder(ctx context.Context, input models.CreateOrderInput) error {
    serviceLogger := logger.WithComponent("order-service")
    serviceLogger.Info("Creating new order", "customer_name", input.CustomerName)
    
    // ... business logic
    
    serviceLogger.WithFields(map[string]interface{}{
        "customer_name": order.CustomerName,
        "total_amount": order.TotalAmount,
        "item_count": len(items),
    }).Debug("Order prepared for creation")
    
    err := s.repo.CreateOrder(ctx, order, items)
    if err != nil {
        serviceLogger.WithError(err).Error("Failed to create order in repository")
        return err
    }
    
    serviceLogger.Info("Order created successfully")
    return nil
}
```

### 6. Database Operations

```go
func (r *OrderRepository) CreateOrder(ctx context.Context, order models.Order, items []models.OrderItem) error {
    repoLogger := logger.WithComponent("order-repository")
    start := time.Now()
    
    repoLogger.Debug("Starting database transaction")
    
    tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
    if err != nil {
        repoLogger.WithError(err).Error("Failed to begin transaction")
        return err
    }
    defer tx.Rollback(ctx)
    
    // ... database operations
    
    duration := time.Since(start)
    logger.LogDatabaseQuery(repoLogger, "INSERT INTO orders", duration, "")
    
    repoLogger.Info("Order created in database", "order_id", order.ID)
    return nil
}
```

### 7. Error Handling with Context

```go
func processOrder(orderID int) error {
    processLogger := logger.WithFields(map[string]interface{}{
        "order_id": orderID,
        "operation": "process_order",
    })
    
    processLogger.Info("Starting order processing")
    
    order, err := getOrder(orderID)
    if err != nil {
        processLogger.WithError(err).Error("Failed to retrieve order")
        return err
    }
    
    if order.Status != "pending" {
        processLogger.Warn("Order is not in pending status", "current_status", order.Status)
        return errors.New("order cannot be processed")
    }
    
    processLogger.Info("Order processing completed successfully")
    return nil
}
```

## HTTP Request Logging

The middleware automatically logs:
- Request start and completion
- Request ID (auto-generated or from X-Request-ID header)
- HTTP method and path
- Status code and response time
- Client IP and User-Agent
- Request/response correlation

Example log output:
```json
{
  "time": "2025-06-20T10:30:45.123Z",
  "level": "INFO",
  "msg": "HTTP request processed",
  "request_id": "req-12345",
  "method": "POST",
  "path": "/api/v1/orders",
  "status_code": 201,
  "duration_ms": 45,
  "remote_ip": "192.168.1.100",
  "user_agent": "Mozilla/5.0...",
  "type": "http_request"
}
```

## Best Practices

1. **Always use structured logging** with key-value pairs instead of formatted strings
2. **Use appropriate log levels**:
   - `Debug`: Detailed information for debugging
   - `Info`: General information about application flow
   - `Warn`: Warning conditions that should be noted
   - `Error`: Error conditions that need attention

3. **Include relevant context** like user IDs, request IDs, resource IDs
4. **Use component-based loggers** to identify the source of log messages
5. **Log errors with context** using `WithError(err)` for error tracking
6. **Log performance metrics** for database queries and external API calls
7. **Use request ID** to trace requests across multiple services

## Request ID Tracing

Request IDs are automatically:
- Generated for each HTTP request
- Included in all log messages within that request
- Returned in the `X-Request-ID` response header
- Available in Fiber context via `c.Locals("request_id")`

This enables complete request tracing across all components of your application.

## Performance Considerations

- The logger is designed to be performant with minimal overhead
- JSON format is recommended for production for better parsing
- Use appropriate log levels to avoid excessive logging in production
- Consider using file output for production deployments

## Monitoring and Alerting

With structured logging, you can easily:
- Query logs by request ID, component, or any custom field
- Set up alerts based on error rates or specific error patterns
- Create dashboards for application metrics
- Integrate with log aggregation systems like ELK, Splunk, or cloud logging services
