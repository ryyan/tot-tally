CREATE TABLE babies (
    id TEXT PRIMARY KEY NOT NULL,
    name TEXT NOT NULL,
    timezone TEXT NOT NULL
);

CREATE TABLE feeds (
    id TEXT PRIMARY KEY NOT NULL,
    baby_id TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL,
    note TEXT NOT NULL,
    ounces INTEGER NOT NULL,
    FOREIGN KEY (baby_id) REFERENCES babies(id) ON DELETE CASCADE
);

CREATE TABLE soils(
    id TEXT PRIMARY KEY NOT NULL,
    baby_id TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL,
    note TEXT NOT NULL,
    wet INTEGER NOT NULL,
    soil INTEGER NOT NULL,
    FOREIGN KEY (baby_id) REFERENCES babies(id) ON DELETE CASCADE
);