CREATE TABLE notes (
    id INT AUTO_INCREMENT PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    content TEXT NOT NULL,
    category VARCHAR(100),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_category (category),
    INDEX idx_created_at (created_at)
);

-- Add user_id to notes table
ALTER TABLE notes ADD COLUMN user_id INT NOT NULL DEFAULT 1;

-- Add foreign key constraint
ALTER TABLE notes ADD CONSTRAINT fk_notes_user_id 
FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

-- Add index for better performance
ALTER TABLE notes ADD INDEX idx_user_id (user_id);

-- Insert default admin user (password: admin123)
INSERT INTO users (username, email, password_hash, full_name) 
VALUES ('admin', 'admin@example.com', '$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi', 'Administrator')
ON DUPLICATE KEY UPDATE username=username;