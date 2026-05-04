SELECT
    a.id,
    a.generation,
    a.network,
    a.dns,
    a.hardware_address,
    a.lat,
    a.lng,
    a.city,
    a.country,
    a.country_code
FROM
    event_device a
WHERE
    a.address = $1
ORDER BY
    a.generation DESC