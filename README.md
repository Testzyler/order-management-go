# Order Management Service (Go)

This is a sample order management service built with Go. It provides a RESTful API for creating, reading, updating, and deleting orders. The project is structured following Clean Architecture principles to ensure separation of concerns, maintainability, and testability.

## Features

- **CRUD Operations:** Full support for creating, reading, updating, and deleting orders.
- **RESTful API:** A well-defined API built with the [Fiber](https://gofiber.io/) web framework.
- **Configuration Management:** Flexible configuration using [Viper](https://github.com/spf13/viper), supporting both file and environment variables.
- **Database Integration:** Uses PostgreSQL as the database, with `sqlx` for easier database interactions.
- **Command-Line Interface:** Powered by [Cobra](https://github.com/spf13/cobra) for a robust CLI experience.
- **Graceful Shutdown:** Handles termination signals to shut down the server gracefully.
- **Context Propagation:** Manages request context for timeouts and cancellations.
- **Containerized:** Includes a `docker-compose.yaml` for easy setup of the PostgreSQL database.

## Tech Stack

- **Language:** [Go](https://golang.org/)
- **Web Framework:** [Fiber](https://gofiber.io/)
- **Database:** [PostgreSQL](https://www.postgresql.org/)
- **CLI:** [Cobra](https://github.com/spf13/cobra)
- **Configuration:** [Viper](https://github.com/spf13/viper)
- **Containerization:** [Docker](https://www.docker.com/) & [Docker Compose](https://docs.docker.com/compose/)

## Prerequisites

- [Go](https://golang.org/doc/install) (version 1.18 or higher)
- [Docker](https://docs.docker.com/get-docker/)
- [Docker Compose](https://docs.docker.com/compose/install/)

## Getting Started

### 1. Clone the Repository

```bash
git clone <repository-url>
cd order-management-go
```

### 2. Configure the Application

Copy the example configuration file and update it with your database credentials if needed.

```bash
cp config/config.example.yaml config/config.yaml
```

The default configuration (`config/config.yaml`) is set up to work with the provided Docker Compose setup.

### 3. Start the Database

Run the PostgreSQL database in a Docker container using Docker Compose.

```bash
docker-compose up -d
```

This will start a PostgreSQL server on `localhost:5432`.

### 4. Initialize the Database

Run the `init.sql` script to create the necessary tables and schema.

```bash
psql -h localhost -p 5432 -U dborder -d store -f init.sql
```
*Password is `SecretP@ssw0rd` as defined in `config.yaml` and `docker-compose.yaml`.*

### 5. Run the Application

Start the HTTP server using the following command:

```bash
go run . http-serve
```

The server will start on `http://localhost:3333`.

## API Endpoints

The following endpoints are available under the `/orders` prefix:

| Method | Path              | Description                               |
|--------|-------------------|-------------------------------------------|
| `POST` | `/`               | Create a new order.                       |
| `GET`    | `/`               | List all orders with pagination.          |
| `GET`    | `/:id`            | Get a single order by its ID.             |
| `PUT`    | `/:id`            | Update an existing order.                 |
| `DELETE` | `/:id`            | Delete an order by its ID.                |

### Example Usage (cURL)

**List Orders (with pagination):**
```bash
curl "http://localhost:3333/orders?page=1&size=5"
```

**Create an Order:**
```bash
curl -X POST http://localhost:3333/orders \
-H "Content-Type: application/json" \
-d '{"customer_name": "John Doe", "total_amount": 199.99, "status": "pending"}'
```

**Get an Order:**
```bash
curl http://localhost:3333/orders/1
```

## Stress Testing

This project includes a command to run a stress test against the `CreateOrder` endpoint. This helps in evaluating the performance and stability of the service under a high load.

To run the stress test, use the following command:

```bash
go run . stress-test --num 17000 --batch 3 --concurrency 50
```

### Flags

- `--num`: The total number of orders to create.
- `--batch`: The number of orders to create in a single batch request.
- `--concurrency`: The number of concurrent workers sending requests.

## Project Structure

The project follows a Clean Architecture-like structure:

- `application/`: Contains the core business logic, including domain models, services (use cases), and repository interfaces.
- `cmd/`: Contains the command-line interface logic using Cobra. This is the entry point of the application.
- `config/`: Configuration files.
- `infrastructure/`: Contains implementations of external concerns like the database, HTTP server, and other third-party integrations.
- `main.go`: The main function that executes the root command.
- `init.sql`: SQL script for database schema initialization.
- `docker-compose.yaml`: Defines the services for the development environment (e.g., database).

