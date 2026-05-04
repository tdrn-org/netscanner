SELECT
    a.id,
    a.count,
    a.first,
    a.last
FROM
    event_action a
WHERE
    a.target_id = $1 AND
    a.device_id = $2 AND
    a.user = $3 AND
    a.status = $4