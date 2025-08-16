-- CreateTable
CREATE TABLE "user_account" (
    "id" BIGSERIAL NOT NULL,
    "user_external_id" TEXT NOT NULL,
    "stripe_subscription_id" VARCHAR(255),
    "stripe_plan_id" VARCHAR(255),
    "stripe_customer_id" VARCHAR(255),
    "created_at" BIGINT NOT NULL DEFAULT ((extract(epoch from now()) * 1000))::bigint,
    "updated_at" BIGINT NOT NULL DEFAULT ((extract(epoch from now()) * 1000))::bigint,

    CONSTRAINT "user_account_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "invalid_subscription" (
    "id" BIGSERIAL NOT NULL,
    "user_external_id" TEXT NOT NULL,
    "stripe_subscription_id" VARCHAR(255),
    "stripe_plan_id" VARCHAR(255),
    "stripe_customer_id" VARCHAR(255),
    "created_at" BIGINT NOT NULL DEFAULT ((extract(epoch from now()) * 1000))::bigint,
    "updated_at" BIGINT NOT NULL DEFAULT ((extract(epoch from now()) * 1000))::bigint,

    CONSTRAINT "invalid_subscription_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "free_credit" (
    "id" BIGSERIAL NOT NULL,
    "user_external_id" TEXT NOT NULL,
    "credit" INTEGER NOT NULL,
    "created_at" BIGINT NOT NULL DEFAULT ((extract(epoch from now()) * 1000))::bigint,
    "updated_at" BIGINT NOT NULL DEFAULT ((extract(epoch from now()) * 1000))::bigint,

    CONSTRAINT "free_credit_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "spending_unit" (
    "id" BIGSERIAL NOT NULL,
    "external_id" TEXT NOT NULL,
    "user_external_id" TEXT NOT NULL,
    "amount" INTEGER NOT NULL DEFAULT 1,
    "created_at" BIGINT NOT NULL DEFAULT ((extract(epoch from now()) * 1000))::bigint,
    "updated_at" BIGINT NOT NULL DEFAULT ((extract(epoch from now()) * 1000))::bigint,

    CONSTRAINT "spending_unit_pkey" PRIMARY KEY ("id")
);

-- CreateIndex
CREATE UNIQUE INDEX "user_account_user_external_id_key" ON "user_account"("user_external_id");

-- CreateIndex
CREATE INDEX "invalid_subscription_user_external_id_idx" ON "invalid_subscription"("user_external_id");

-- CreateIndex
CREATE UNIQUE INDEX "free_credit_user_external_id_key" ON "free_credit"("user_external_id");

-- CreateIndex
CREATE UNIQUE INDEX "spending_unit_external_id_key" ON "spending_unit"("external_id");

-- CreateIndex
CREATE INDEX "spending_unit_user_external_id_idx" ON "spending_unit"("user_external_id");

-- CreateIndex
CREATE INDEX "spending_unit_created_at_idx" ON "spending_unit"("created_at");

-- AddForeignKey
ALTER TABLE "invalid_subscription" ADD CONSTRAINT "invalid_subscription_user_external_id_fkey" FOREIGN KEY ("user_external_id") REFERENCES "user_account"("user_external_id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "free_credit" ADD CONSTRAINT "free_credit_user_external_id_fkey" FOREIGN KEY ("user_external_id") REFERENCES "user_account"("user_external_id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "spending_unit" ADD CONSTRAINT "spending_unit_user_external_id_fkey" FOREIGN KEY ("user_external_id") REFERENCES "user_account"("user_external_id") ON DELETE CASCADE ON UPDATE CASCADE;

