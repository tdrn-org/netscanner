SELECT
    a.id,
    a.name,
    a.tokenizer
FROM
    log_matcher a
WHERE
    a.profile_id = $1