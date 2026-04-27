UPDATE log_matcher_index
SET
    version = $1
WHERE
    name = $2