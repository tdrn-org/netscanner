SELECT
    a.address,
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
    device a
WHERE
    a.id = $1
ORDER BY
    a.generation DESC