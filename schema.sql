CREATE TABLE babies (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    timezone TEXT NOT NULL
);

CREATE TABLE feeds (
    id INTEGER PRIMARY KEY,
    baby_id TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL,
    note TEXT NOT NULL,
    ounces INTEGER NOT NULL,
    FOREIGN KEY (baby_id) REFERENCES babies(id) ON DELETE CASCADE
);

CREATE TABLE soils(
    id INTEGER PRIMARY KEY,
    baby_id TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL,
    note TEXT NOT NULL,
    wet TEXT NOT NULL,
    soil TEXT NOT NULL,
    FOREIGN KEY (baby_id) REFERENCES babies(id) ON DELETE CASCADE
);