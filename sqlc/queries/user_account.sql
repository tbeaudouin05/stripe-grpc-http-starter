-- name: GetSubscriptionIDByUserExternalID :one
SELECT stripe_subscription_id
FROM user_account
WHERE user_external_id = $1;

-- name: UpsertUserAccount :exec
INSERT INTO user_account (
  user_external_id,
  stripe_subscription_id,
  stripe_plan_id,
  stripe_customer_id
) VALUES ($1, $2, $3, $4)
ON CONFLICT (user_external_id) DO UPDATE SET
  stripe_subscription_id = COALESCE(EXCLUDED.stripe_subscription_id, user_account.stripe_subscription_id),
  stripe_plan_id = COALESCE(EXCLUDED.stripe_plan_id, user_account.stripe_plan_id),
  stripe_customer_id = COALESCE(EXCLUDED.stripe_customer_id, user_account.stripe_customer_id);

-- name: GetUserAccount :one
SELECT 
  user_external_id,
  stripe_subscription_id,
  stripe_plan_id,
  stripe_customer_id,
  created_at,
  updated_at
FROM user_account
WHERE user_external_id = $1;
