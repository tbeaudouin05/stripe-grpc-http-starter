-- name: UpsertAndGetFreeCredit :one
INSERT INTO free_credit (
  user_external_id,
  credit
) VALUES ($1, $2)
ON CONFLICT (user_external_id) DO UPDATE SET user_external_id = EXCLUDED.user_external_id
RETURNING credit;
