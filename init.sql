CREATE SCHEMA IF NOT EXISTS store;
CREATE TABLE
    store.orders (
        id SERIAL PRIMARY KEY,
        customer_name VARCHAR(100),
        total_amount DECIMAL(10, 2),
        status VARCHAR(50),
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    );

CREATE TABLE
    store.order_items (
        id SERIAL PRIMARY KEY,
        order_id INT REFERENCES store.orders (id) ON DELETE CASCADE,
        product_name VARCHAR(100),
        quantity INT,
        price DECIMAL(10, 2),
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    );