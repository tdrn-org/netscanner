UPDATE
    connection
SET
    count = $1,
    last = $2
WHERE
    id = $3