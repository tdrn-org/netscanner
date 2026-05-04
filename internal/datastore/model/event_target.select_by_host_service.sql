SELECT
    a.id
FROM
    event_target a
WHERE
    a.host = $1 AND
    a.service = $2