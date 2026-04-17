--
-- Profile
--
CREATE TABLE profile(
    id TEXT NOT NULL,
    name TEXT NOT NULL,
    PRIMARY KEY(id),
    UNIQUE(name)
);
CREATE TABLE log_matcher(
    id TEXT NOT NULL,
    profile_id TEXT NOT NULL,
    name TEXT NOT NULL,
    tokenizer TEXT NOT NULL,
    PRIMARY KEY(id),
    FOREIGN KEY(profile_id) REFERENCES profile(id)
);
CREATE TABLE log_matcher_entry(
    id TEXT NOT NULL,
    log_matcher_id TEXT NOT NULL,
    service TEXT NOT NULL,
    event_type TEXT NOT NULL,
    match TEXT NOT NULL,
    PRIMARY KEY(id),
    FOREIGN KEY(log_matcher_id) REFERENCES log_matcher(id)
);
CREATE TABLE syslog_sensor(
    id TEXT NOT NULL,
    profile_id TEXT NOT NULL,
    name TEXT NOT NULL,
    enabled INTEGER NOT NULL,
    network TEXT NOT NULL,
    address TEXT NOT NULL,
    log_matcher TEXT NOT NULL,
    PRIMARY KEY(id),
    FOREIGN KEY(profile_id) REFERENCES profile(id)
);
--
-- EOF
--
