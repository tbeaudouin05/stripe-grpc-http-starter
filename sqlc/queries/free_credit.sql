-- name: UpsertAndGetFreeCredit :one
INSERT INTO free_credit (
  user_external_id,
  credit
) VALUES ($1, $2)
ON CONFLICT (user_external_id) DO UPDATE SET user_external_id = EXCLUDED.user_external_id
RETURNING credit;

-- name: ConsumeFreeCredit :exec
UPDATE free_credit
SET credit = credit - LEAST(credit, $2)
WHERE user_external_id = $1;
