CREATE SCHEMA IF NOT EXISTS public;
ALTER ROLE appusr SET search_path TO public;
SET search_path TO public;
-- members table
CREATE TABLE IF NOT EXISTS members (
    id          SERIAL PRIMARY KEY,
    username    VARCHAR(255) NOT NULL,
    email       VARCHAR(255) UNIQUE NOT NULL,
    member_id   VARCHAR(255) UNIQUE NOT NULL,
    customer_id VARCHAR(255),
    created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_by  VARCHAR(255) DEFAULT 'system',
    updated_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_by  VARCHAR(255) DEFAULT 'system',
    deleted     BOOLEAN DEFAULT FALSE,
    deleted_at  TIMESTAMP,
    deleted_by  VARCHAR(255),
    status_acc  VARCHAR(50) NOT NULL DEFAULT 'PENDING'
);

-- counter table for member_id generation
CREATE TABLE IF NOT EXISTS member_id_counters (
    date     DATE PRIMARY KEY,
    last_seq INT NOT NULL DEFAULT 0
);

ALTER TABLE members ADD COLUMN IF NOT EXISTS role VARCHAR(50) NOT NULL DEFAULT 'member';

CREATE TABLE IF NOT EXISTS role_permissions (
    id           SERIAL PRIMARY KEY,
    role         VARCHAR(50) NOT NULL,
    method       VARCHAR(10) NOT NULL DEFAULT 'POST',
    path_pattern VARCHAR(255) NOT NULL DEFAULT '/api/v1/members/login'
);


INSERT INTO members (username,email,member_id)
VALUES ('GG','GG%','user-123');

INSERT INTO members (username,email,member_id)
VALUES ('KK','KK%','user-456');

INSERT INTO members (username,email,member_id)
VALUES ('JJ','KJ%','user-789');

INSERT INTO role_permissions (role, method, path_pattern)
VALUES ('admin', 'POST', '/api/v1/members%');
INSERT INTO role_permissions (role)
VALUES ('admin');
INSERT INTO role_permissions (role)
VALUES ('member');
INSERT INTO role_permissions (role, method, path_pattern)
VALUES ('admin', 'GET', '/api/v1/members%');
UPDATE members SET role = 'admin' WHERE member_id = 'user-123';
