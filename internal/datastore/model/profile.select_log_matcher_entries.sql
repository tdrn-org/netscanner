SELECT
    a.id,
    a.service,
    a.event_type,
    a.match
FROM
    log_matcher_entry a
WHERE
    a.log_matcher_id = $1