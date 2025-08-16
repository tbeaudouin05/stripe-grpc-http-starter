-- name: CountUnitsBetween :one
SELECT COALESCE(SUM(amount), 0) AS count
FROM spending_unit
WHERE user_external_id = $1
  AND created_at >= $2
  AND created_at <= $3;

-- name: InsertSpendingUnit :one
WITH ins AS (
    INSERT INTO spending_unit (
        external_id,
        user_external_id,
        amount,
        created_at,
        updated_at
    ) VALUES ($1, $2, $3, $4, $4)
    ON CONFLICT (external_id) DO NOTHING
    RETURNING 1::int AS inserted
)
SELECT COALESCE(SUM(inserted), 0) AS inserted FROM ins;
