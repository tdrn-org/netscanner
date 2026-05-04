UPDATE
    event_action
SET
    count = $1,
    last = $2
WHERE
    id = $3