SELECT
    a.id,
    a.service,
    a.status,
    a.user,
    a.count,
    a.first,
    a.last,
    b.id,
    b.address,
    b.generation,
    b.network,
    b.dns,
    b.hardware_address,
    b.lat,
    b.lng,
    b.city,
    b.country,
    b.country_code,
    c.id,
    c.address,
    c.generation,
    c.network,
    c.dns,
    c.hardware_address,
    c.lat,
    c.lng,
    c.city,
    c.country,
    c.country_code
FROM
    connection a,
    device b,
    device c
WHERE
    a.server_id = b.id AND
    a.client_id = c.id
ORDER BY
    a.last DESC