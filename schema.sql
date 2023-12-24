CREATE TABLE tots (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    timezone TEXT NOT NULL
);

CREATE TABLE tallies (
    id INTEGER PRIMARY KEY,
    tot_id TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL,
    kind TEXT NOT NULL,
    FOREIGN KEY (tot_id) REFERENCES tots(id) ON DELETE CASCADE
);
