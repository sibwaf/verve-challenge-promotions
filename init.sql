CREATE TABLE promotions_0 (
    id UUID PRIMARY KEY,
    table_id INT NOT NULL DEFAULT 0,
    price DOUBLE NOT NULL,
    expiration_date DATETIME NOT NULL
);

CREATE TABLE promotions_1 (
    id UUID PRIMARY KEY,
    table_id INT NOT NULL DEFAULT 1,
    price DOUBLE NOT NULL,
    expiration_date DATETIME NOT NULL
);

CREATE TABLE db_state (
    property_name TEXT NOT NULL,
    property_value TEXT NOT NULL
);

INSERT INTO db_state (property_name, property_value) VALUES ('current_table_id', '0');
