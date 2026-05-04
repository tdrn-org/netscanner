--
-- Events
--
CREATE TABLE event_target(
    id TEXT NOT NULL,
    host TEXT NOT NULL,
    service TEXT NOT NULL,
    PRIMARY KEY(id),
    UNIQUE(host, service)
);
CREATE TABLE event_device(
    id TEXT NOT NULL,
    address TEXT NOT NULL,
    generation BIGINT NOT NULL,
    network TEXT NOT NULL,
    dns TEXT NOT NULL,
    hardware_address TEXT NOT NULL,
    lat DOUBLE PRECISION,
    lng DOUBLE PRECISION,
    city TEXT NOT NULL,
    country TEXT NOT NULL,
    country_code TEXT NOT NULL,
    PRIMARY KEY(id),
    UNIQUE(address, generation)
);
CREATE TABLE event_action(
    id TEXT NOT NULL,
    target_id TEXT NOT NULL,
    device_id TEXT NOT NULL,
    user TEXT NOT NULL,
    status TEXT NOT NULL,
    count BIGINT NOT NULL,
    first BIGINT NOT NULL,
    last BIGINT NOT NULL,
    PRIMARY KEY(id),
    UNIQUE(target_id,device_id,user,status)
);
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
-- Network
--
CREATE TABLE network(
    name TEXT NOT NULL,
    version INTEGER NOT NULL,
    PRIMARY KEY(name)
);
CREATE TABLE network_entry(
    network_name TEXT NOT NULL,
    cidr TEXT NOT NULL,
    FOREIGN KEY(network_name) REFERENCES network(name)
);
--
-- EOF
--
INSERT INTO log_matcher_index(name,version) VALUES('syslog1',1);
INSERT INTO log_matcher_index_entry(log_matcher_index_name,service,event_type,match) VALUES('syslog1','sshd','denied','Connection closed by authenticating user {User} {IP} port');
INSERT INTO log_matcher_index_entry(log_matcher_index_name,service,event_type,match) VALUES('syslog1','sshd','denied','Invalid user {User} from {IP} port');
INSERT INTO log_matcher_index_entry(log_matcher_index_name,service,event_type,match) VALUES('syslog1','sshd','denied','Connection reset by authenticating user {User} {IP} port');
INSERT INTO log_matcher_index_entry(log_matcher_index_name,service,event_type,match) VALUES('syslog1','sshd','error','Timeout before authentication for connection from {IP} to {Any}');
INSERT INTO log_matcher_index_entry(log_matcher_index_name,service,event_type,match) VALUES('syslog1','sshd','granted','Accepted publickey for {User} from {IP} port {Any} ssh2: RSA');
