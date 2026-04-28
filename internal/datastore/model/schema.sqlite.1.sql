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
INSERT INTO log_matcher_index(name,version) VALUES('syslog1',1);
INSERT INTO log_matcher_index_entry(log_matcher_index_name,service,event_type,match) VALUES('syslog1','sshd','denied','Connection reset by authenticating user {User} {IP} port {Any} [preauth]');
INSERT INTO log_matcher_index_entry(log_matcher_index_name,service,event_type,match) VALUES('syslog1','sshd','denied','Accepted publickey for {User} from {IP} port {Any} ssh2: RSA');
