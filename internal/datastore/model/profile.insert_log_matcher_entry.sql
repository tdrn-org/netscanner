INSERT INTO
    log_matcher_entry(
        id,
        log_matcher_id,
        service,
        event_type,
        match
    )
VALUES(
    $1,
    $2,
    $3,
    $4,
    $5
)