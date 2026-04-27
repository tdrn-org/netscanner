SELECT
    a.version
FROM
    log_matcher_index a
WHERE
    a.name = $1