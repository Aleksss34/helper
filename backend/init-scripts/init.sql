CREATE TABLE users(
      id SERIAL PRIMARY KEY,
      username VARCHAR(30) NOT NULL,
      email VARCHAR(100) NOT NULL,
      hash_password VARCHAR(64) NOT NULL,
      status VARCHAR(20) NOT NULL,
      count_questions INTEGER,
      CONSTRAINT idx_users_username UNIQUE (username),
      CONSTRAINT idx_users_email UNIQUE (email)
);
