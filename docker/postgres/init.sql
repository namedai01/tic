-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Create initial admin user (optional)
-- This will be created by the application, but you can pre-create users here if needed

-- Create indexes for better performance
-- These will be created by GORM, but you can add custom indexes here
