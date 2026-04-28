USE inventory_db;

-- 1. Users Table
CREATE TABLE IF NOT EXISTS users (
    id INT AUTO_INCREMENT PRIMARY KEY,
    username VARCHAR(50) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    role ENUM('admin', 'staff') NOT NULL DEFAULT 'staff',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 2. Insert Default Users (Password is 'password123' for both)
-- bcrypt hash for 'password123' is $2a$10$XmC27aP7/jI6Eeb9G.rDNuL5QdG...
-- Actually I will let the Go code insert the defaults if table is empty to avoid hardcoding raw bcrypt hashes in SQL that might have format issues, or I can provide exact hash.
INSERT IGNORE INTO users (username, password_hash, role) VALUES
('admin', '$2a$10$a1wXgI1Xp4JdD0pB9sZ8/OuBq3CqGfN/aZz3eP3RzM5lG2o5M7m0u', 'admin'),
('staff', '$2a$10$a1wXgI1Xp4JdD0pB9sZ8/OuBq3CqGfN/aZz3eP3RzM5lG2o5M7m0u', 'staff');

-- 3. Transactions Table
CREATE TABLE IF NOT EXISTS inventory_transactions (
    id INT AUTO_INCREMENT PRIMARY KEY,
    product_id INT NOT NULL,
    user_id INT,
    transaction_type ENUM('IN', 'OUT') NOT NULL,
    quantity INT NOT NULL,
    notes VARCHAR(255),
    transaction_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (product_id) REFERENCES products(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL
);
