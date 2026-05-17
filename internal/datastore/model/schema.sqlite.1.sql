--
-- Devices
--
CREATE TABLE device(
    id TEXT NOT NULL,
    generation INTEGER NOT NULL,
    address TEXT NOT NULL,
    network TEXT NOT NULL,
    dns TEXT NOT NULL,
    hardware_address TEXT NOT NULL,
    lat REAL NOT NULL,
    lng REAL NOT NULL,
    city TEXT NOT NULL,
    country TEXT NOT NULL,
    country_code TEXT NOT NULL,
    PRIMARY KEY(id),
    UNIQUE(address, generation)
);
--
-- Connections
--
CREATE TABLE connection(
    id TEXT NOT NULL,
    server_id TEXT NOT NULL,
    client_id TEXT NOT NULL,
    service TEXT NOT NULL,
    status TEXT NOT NULL,
    user TEXT NOT NULL,
    count INTEGER NOT NULL,
    first INTEGER NOT NULL,
    last INTEGER NOT NULL,
    PRIMARY KEY(id),
    UNIQUE(server_id,client_id,service,status,user),
    FOREIGN KEY(server_id) REFERENCES device(id),
    FOREIGN KEY(client_id) REFERENCES device(id)
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
