-- name: InsertInvalidSubscription :exec
INSERT INTO invalid_subscription (
  user_external_id,
  stripe_subscription_id,
  stripe_plan_id,
  stripe_customer_id
) VALUES ($1, $2, $3, $4);
