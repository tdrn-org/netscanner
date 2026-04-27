--
-- LogMatcher
--
CREATE TABLE log_matcher_index(
    name TEXT NOT NULL,
    version INTEGER NOT NULL,
    PRIMARY KEY(name)
);
CREATE TABLE log_matcher_index_entry(
    log_matcher_index_name TEXT NOT NULL,
    service TEXT NOT NULL,
    event_type TEXT NOT NULL,
    match TEXT NOT NULL,
    FOREIGN KEY(log_matcher_index_name) REFERENCES log_matcher_index(name)
);
--
-- EOF
--
