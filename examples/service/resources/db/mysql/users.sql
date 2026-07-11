-- Create the R/W user
CREATE USER IF NOT EXISTS 'dbrw_user'@'%' IDENTIFIED BY 'dbrw_pass';
-- Grant read and write permissions (SELECT, INSERT, UPDATE, DELETE) on all tables
GRANT SELECT, INSERT, UPDATE, DELETE ON nuragoexample.* TO 'dbrw_user'@'%';

-- Create the R/O user
CREATE USER IF NOT EXISTS 'dbro_user'@'%' IDENTIFIED BY 'dbro_pass';
-- Grant read and write permissions (SELECT, INSERT, UPDATE, DELETE) on all tables
GRANT SELECT ON nuragoexample.* TO 'dbro_user'@'%';

-- Apply the changes
FLUSH PRIVILEGES;