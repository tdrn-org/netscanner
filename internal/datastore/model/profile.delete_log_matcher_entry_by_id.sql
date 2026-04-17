DELETE FROM
    log_matcher_entry a
WHERE
    a.log_matcher_id IN (
        SELECT
            id
        FROM
            log_matcher b
        WHERE
            b.profile_id = $1
    )