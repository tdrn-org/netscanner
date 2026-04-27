SELECT
    a.service,
    a.event_type,
    a.match
FROM
    log_matcher_index_entry a
WHERE
    a.log_matcher_index_name = $1