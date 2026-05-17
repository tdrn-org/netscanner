SELECT
    a.id,
    a.count,
    a.first,
    a.last
FROM
    connection a
WHERE
    a.server_id = $1 AND
    a.client_id = $2 AND
    a.service = $3 AND
    a.status = $4 AND
    a.user = $5