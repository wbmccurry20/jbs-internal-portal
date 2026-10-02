BEGIN;

CREATE TEMP TABLE authoritative_state_licenses (
    state CHAR(2) NOT NULL,
    license_number VARCHAR(100),
    entity_name VARCHAR(255),
    expiration_date DATE,
    application_status VARCHAR(50),
    notes TEXT
) ON COMMIT DROP;

INSERT INTO authoritative_state_licenses
    (state, license_number, entity_name, expiration_date, application_status, notes)
VALUES
    ('SC', 'CLG.125936 GC', NULL, '2026-10-31', NULL, NULL),
    ('CA', '1127453', 'JBS', '2026-10-31', NULL, NULL),
    ('NC', '101877', NULL, '2026-12-31', NULL, NULL),
    ('VA', '2705193033', NULL, '2026-12-31', NULL, NULL),
    ('TN', '82676', NULL, '2027-02-28', NULL, NULL),
    ('WA', '605 423 230', NULL, '2027-03-11', NULL, NULL),
    ('AL', '60614', NULL, '2027-03-31', NULL, NULL),
    ('OR', '256401', NULL, '2027-05-28', NULL, NULL),
    ('WV', 'WV063259', NULL, '2027-06-09', NULL, NULL),
    ('AR', '0456250725', NULL, '2027-07-31', NULL, 'The main license table lists 2027-07-31; the separate renewal notes list 2026-07-31 as submitted.'),
    ('WI', 'NO. 1530 - DCFR', NULL, '2027-09-17', NULL, NULL),
    ('UT', '13820980-5501', NULL, '2027-11-30', NULL, NULL),
    ('ID', '6971088', NULL, '2027-12-08', NULL, NULL),
    ('MN', 'IR812553', NULL, '2027-12-31', NULL, NULL),
    ('AZ', 'ROC 349061', NULL, '2027-12-31', NULL, NULL),
    ('NV', '0094969', NULL, '2028-02-28', NULL, NULL),
    ('FL', 'CGC1536416', NULL, '2028-08-31', NULL, NULL),
    ('NM', '422571', NULL, '2028-10-31', NULL, NULL),
    ('LA', '1828', NULL, '2029-03-19', NULL, NULL),
    ('MS', '200-12540-9', NULL, NULL, NULL, NULL),
    ('GA', NULL, NULL, NULL, 'pending', 'Application in review'),
    ('CA', NULL, 'AJK', NULL, 'pending', 'Application in progress');

UPDATE state_licenses AS current_license
SET license_type = 'State License',
    license_number = source.license_number,
    entity_name = source.entity_name,
    expiration_date = source.expiration_date,
    status = CASE
        WHEN source.application_status IS NOT NULL THEN source.application_status
        WHEN source.expiration_date IS NULL THEN 'active'
        WHEN source.expiration_date < CURRENT_DATE THEN 'expired'
        WHEN source.expiration_date <= CURRENT_DATE + 90 THEN 'expiring'
        ELSE 'active'
    END,
    notes = source.notes,
    city = NULL,
    is_city_license = FALSE,
    is_active = TRUE,
    updated_at = CURRENT_TIMESTAMP
FROM authoritative_state_licenses AS source
WHERE current_license.state = source.state
  AND current_license.city IS NULL
  AND COALESCE(current_license.is_city_license, FALSE) = FALSE
  AND (
      (source.license_number IS NOT NULL AND current_license.license_number = source.license_number)
      OR (
          source.license_number IS NULL
          AND current_license.license_number IS NULL
          AND current_license.entity_name IS NOT DISTINCT FROM source.entity_name
          AND (source.entity_name IS NOT NULL OR current_license.notes = source.notes)
      )
  );

INSERT INTO state_licenses
    (state, license_type, license_number, entity_name, expiration_date, status, notes,
     is_city_license, is_active)
SELECT
    source.state,
    'State License',
    source.license_number,
    source.entity_name,
    source.expiration_date,
    CASE
        WHEN source.application_status IS NOT NULL THEN source.application_status
        WHEN source.expiration_date IS NULL THEN 'active'
        WHEN source.expiration_date < CURRENT_DATE THEN 'expired'
        WHEN source.expiration_date <= CURRENT_DATE + 90 THEN 'expiring'
        ELSE 'active'
    END,
    source.notes,
    FALSE,
    TRUE
FROM authoritative_state_licenses AS source
WHERE NOT EXISTS (
    SELECT 1
    FROM state_licenses AS current_license
    WHERE current_license.state = source.state
      AND current_license.city IS NULL
      AND COALESCE(current_license.is_city_license, FALSE) = FALSE
      AND (
          (source.license_number IS NOT NULL AND current_license.license_number = source.license_number)
          OR (
              source.license_number IS NULL
              AND current_license.license_number IS NULL
              AND current_license.entity_name IS NOT DISTINCT FROM source.entity_name
              AND (source.entity_name IS NOT NULL OR current_license.notes = source.notes)
          )
      )
);

WITH ranked_matches AS (
    SELECT
        current_license.id,
        ROW_NUMBER() OVER (
            PARTITION BY source.state, source.license_number, source.entity_name, source.notes
            ORDER BY (
                SELECT COUNT(*)
                FROM license_documents AS document
                WHERE document.license_id = current_license.id
            ) DESC, current_license.id
        ) AS match_number
    FROM state_licenses AS current_license
    JOIN authoritative_state_licenses AS source
      ON current_license.state = source.state
     AND current_license.city IS NULL
     AND COALESCE(current_license.is_city_license, FALSE) = FALSE
     AND (
         (source.license_number IS NOT NULL AND current_license.license_number = source.license_number)
         OR (
             source.license_number IS NULL
             AND current_license.license_number IS NULL
             AND current_license.entity_name IS NOT DISTINCT FROM source.entity_name
             AND (source.entity_name IS NOT NULL OR current_license.notes = source.notes)
         )
     )
)
DELETE FROM state_licenses AS current_license
USING ranked_matches
WHERE current_license.id = ranked_matches.id
  AND ranked_matches.match_number > 1;

DELETE FROM state_licenses AS current_license
WHERE NOT EXISTS (
    SELECT 1
    FROM authoritative_state_licenses AS source
    WHERE current_license.state = source.state
      AND current_license.city IS NULL
      AND COALESCE(current_license.is_city_license, FALSE) = FALSE
      AND (
          (source.license_number IS NOT NULL AND current_license.license_number = source.license_number)
          OR (
              source.license_number IS NULL
              AND current_license.license_number IS NULL
              AND current_license.entity_name IS NOT DISTINCT FROM source.entity_name
              AND (source.entity_name IS NOT NULL OR current_license.notes = source.notes)
          )
      )
);

COMMIT;