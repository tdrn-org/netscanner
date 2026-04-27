INSERT INTO
    log_matcher_index_entry(
        log_matcher_index_name,
        service,
        event_type,
        match
    )
VALUES(
    $1,
    $2,
    $3,
    $4
)