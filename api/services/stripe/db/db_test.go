package db_test

import (
    "crypto/sha256"
    "database/sql"
    "encoding/hex"
    "fmt"
    "testing"

    config "github.com/tbeaudouin05/stripe-trellai/api/config"
    database "github.com/tbeaudouin05/stripe-trellai/api/database"
    stripedb "github.com/tbeaudouin05/stripe-trellai/api/services/stripe/db"
)

func hash(s string) string {
    // Sanitize to ASCII alphanumerics to mirror production hashing
    b := make([]byte, 0, len(s))
    for i := 0; i < len(s); i++ {
        c := s[i]
        if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') {
            b = append(b, c)
        }
    }
    sum := sha256.Sum256(b)
    return hex.EncodeToString(sum[:])
}

func TestMain(m *testing.M) {
    // Prevent tests from running against production database
    config.CheckNotProdDB()

    // Load config and initialize DB
    cfg, err := config.LoadConfig()
    if err != nil {
        panic(err)
    }
    config.AppConfig = cfg
    if err := database.Initialize(); err != nil {
        panic(err)
    }
    // Pre-test cleanup for IDs used in this package
    dbc := database.GetDB()
    ids := []string{"db-test-board", "db-test-ticket-board", "db-test-free-credit", "dup-board", "test-check-board"}
    for _, id := range ids {
        hid := hash(id)
        _, _ = dbc.Exec("DELETE FROM spending_unit WHERE user_external_id = $1", hid)
        _, _ = dbc.Exec("DELETE FROM free_credit WHERE user_external_id = $1", hid)
        _, _ = dbc.Exec("DELETE FROM invalid_subscription WHERE user_external_id = $1", hid)
        _, _ = dbc.Exec("DELETE FROM user_account WHERE user_external_id = $1", hid)
    }
    // Run tests
    m.Run()
}

func TestInsertAndGetUserAccount(t *testing.T) {
    id := "db-test-board"
    hid := hash(id)
    // cleanup
    defer database.GetDB().Exec("DELETE FROM free_credit WHERE user_external_id = $1", hid)
    defer database.GetDB().Exec("DELETE FROM user_account WHERE user_external_id = $1", hid)
    defer database.GetDB().Exec("DELETE FROM invalid_subscription WHERE user_external_id = $1", hid)

    // should not exist yet
    exists, _, err := stripedb.CheckUserAccount(id)
    if err != nil {
        t.Fatalf("CheckUserAccount failed: %v", err)
    }
    if exists {
        t.Fatalf("expected account to not exist, got exists")
    }
    // Upsert new account
    err = stripedb.UpsertUserAccount(id, "sub1", "plan1", "cust1")
    if err != nil {
        t.Fatalf("UpsertUserAccount failed: %v", err)
    }

    // verify via CheckUserAccount
    exists, subID, err := stripedb.CheckUserAccount(id)
    if err != nil || !exists || subID != "sub1" {
        t.Fatalf("CheckUserAccount expected (true, sub1), got (%v, %v), err %v", exists, subID, err)
    }

    // Get and verify
    account, err := stripedb.GetUserAccount(id)
    if err != nil {
        t.Fatalf("GetUserAccount failed: %v", err)
    }
    if account.StripeSubscriptionID != "sub1" {
        t.Errorf("Expected subscription 'sub1', got '%v'", account.StripeSubscriptionID)
    }

    // Second insert goes to invalid_subscription and does not update user_account
    err = stripedb.InsertInvalidSubscription(id, "sub2", "plan2", "cust2")
    if err != nil {
        t.Fatalf("InsertInvalidSubscription failed: %v", err)
    }
    account, err = stripedb.GetUserAccount(id)
    if err != nil {
        t.Fatalf("GetUserAccount failed: %v", err)
    }
    if account.StripeSubscriptionID != "sub1" {
        t.Errorf("Expected subscription to remain 'sub1', got '%v'", account.StripeSubscriptionID)
    }
    // Verify invalid_subscription entry
    var invSub sql.NullString
    err = database.GetDB().QueryRow("SELECT stripe_subscription_id FROM invalid_subscription WHERE user_external_id = $1", hid).Scan(&invSub)
    if err != nil {
        t.Fatalf("Query invalid_subscription failed: %v", err)
    }
    if !invSub.Valid || invSub.String != "sub2" {
        t.Errorf("Expected invalid_subscription 'sub2', got '%v'", invSub.String)
    }
}

func TestFreeCreditLifecycle(t *testing.T) {
    id := "db-test-free-credit"
    hid := hash(id)
    // cleanup
    defer database.GetDB().Exec("DELETE FROM free_credit WHERE user_external_id = $1", hid)
    defer database.GetDB().Exec("DELETE FROM user_account WHERE user_external_id = $1", hid)

    // Ensure account exists for FK
    err := stripedb.UpsertUserAccount(id, "", "", "")
    if err != nil {
        t.Fatalf("UpsertUserAccount for free_credit failed: %v", err)
    }

    // Test initial credit
    credit, err := stripedb.GetFreeCredit(id)
    if err != nil {
        t.Fatalf("GetFreeCredit failed: %v", err)
    }
    if credit != config.AppConfig.InitialFreeCredit {
        t.Errorf("Expected credit %d, got %d", config.AppConfig.InitialFreeCredit, credit)
    }

    // Test updating credit directly in DB
    _, err = database.GetDB().Exec("UPDATE free_credit SET credit = 10 WHERE user_external_id = $1", hid)
    if err != nil {
        t.Fatalf("Failed to update credit directly: %v", err)
    }

    // Verify update
    credit, err = stripedb.GetFreeCredit(id)
    if err != nil {
        t.Fatalf("GetFreeCredit after update failed: %v", err)
    }
    if credit != 10 {
        t.Errorf("Expected credit 10 after update, got %d", credit)
    }
}

func TestCountUnitsBetween(t *testing.T) {
    boardID := "db-test-ticket-board"
    hboard := hash(boardID)
    // cleanup
    defer database.GetDB().Exec("DELETE FROM spending_unit WHERE user_external_id = $1", hboard)
    defer database.GetDB().Exec("DELETE FROM user_account WHERE user_external_id = $1", hboard)

    // Ensure account exists
    err := stripedb.UpsertUserAccount(boardID, "sub", "plan", "cust")
    if err != nil {
        t.Fatalf("UpsertUserAccount failed: %v", err)
    }

    now := int64(1713800000) // fixed timestamp for reproducibility
    // Insert units at different timestamps
    units := []struct {
        externalID string
        createdAt  int64
    }{
        {"db-unit1", now - 100},
        {"db-unit2", now},
        {"db-unit3", now + 100},
    }
    for i, u := range units {
        // Use unique external IDs to avoid duplicate key constraint
        eid := fmt.Sprintf("%s-%d", u.externalID, i)
        _, err := database.GetDB().Exec(`INSERT INTO spending_unit (external_id, user_external_id, amount, created_at, updated_at) VALUES ($1, $2, 1, $3, $3)`,
            eid, hboard, u.createdAt)
        if err != nil {
            t.Fatalf("Failed to insert spending_unit: %v", err)
        }
    }

    // Should count all 3
    count, err := stripedb.CountUnitsBetween(boardID, now-200, now+200)
    if err != nil {
        t.Fatalf("CountUnitsBetween failed: %v", err)
    }
    if count != 3 {
        t.Errorf("Expected 3 units, got %d", count)
    }

    // Should count only 2 (now and after)
    count, err = stripedb.CountUnitsBetween(boardID, now, now+200)
    if err != nil {
        t.Fatalf("CountUnitsBetween failed: %v", err)
    }
    if count != 2 {
        t.Errorf("Expected 2 units, got %d", count)
    }

    // Should count only 1 (exact match)
    count, err = stripedb.CountUnitsBetween(boardID, now+100, now+100)
    if err != nil {
        t.Fatalf("CountUnitsBetween failed: %v", err)
    }
    if count != 1 {
        t.Errorf("Expected 1 unit, got %d", count)
    }
}

func TestDuplicateBoardID(t *testing.T) {
    id := "dup-board"
    hid := hash(id)
    // cleanup
    defer database.GetDB().Exec("DELETE FROM invalid_subscription WHERE user_external_id = $1", hid)
    defer database.GetDB().Exec("DELETE FROM user_account WHERE user_external_id = $1", hid)
    // first upsert
    if err := stripedb.UpsertUserAccount(id, "s1", "p1", "c1"); err != nil {
        t.Fatalf("UpsertUserAccount failed: %v", err)
    }
    // duplicate subscription logged as invalid
    if err := stripedb.InsertInvalidSubscription(id, "s2", "p2", "c2"); err != nil {
        t.Fatalf("InsertInvalidSubscription failed: %v", err)
    }
    // verify user_account unchanged
    account, err := stripedb.GetUserAccount(id)
    if err != nil {
        t.Fatalf("GetUserAccount failed: %v", err)
    }
    if account.StripeSubscriptionID != "s1" || account.StripePlanID != "p1" || account.StripeCustomerID != "c1" {
        t.Errorf("user_account changed on duplicate: got (%v,%v,%v)", account.StripeSubscriptionID, account.StripePlanID, account.StripeCustomerID)
    }
    // verify invalid_subscription entry
    var invSub, invPlan, invCust sql.NullString
    err = database.GetDB().QueryRow(
        "SELECT stripe_subscription_id, stripe_plan_id, stripe_customer_id FROM invalid_subscription WHERE user_external_id = $1", hid,
    ).Scan(&invSub, &invPlan, &invCust)
    if err != nil {
        t.Fatalf("Query invalid_subscription failed: %v", err)
    }
    if invSub.String != "s2" || invPlan.String != "p2" || invCust.String != "c2" {
        t.Errorf("invalid_subscription wrong: got (%v,%v,%v)", invSub.String, invPlan.String, invCust.String)
    }
}

func TestCheckUserAccount(t *testing.T) {
    id := "test-check-board"
    hid := hash(id)
    // cleanup
    defer database.GetDB().Exec("DELETE FROM user_account WHERE user_external_id = $1", hid)

    // should not exist yet
    exists, _, err := stripedb.CheckUserAccount(id)
    if err != nil {
        t.Fatalf("CheckUserAccount failed: %v", err)
    }
    if exists {
        t.Fatalf("expected account to not exist, got exists")
    }

    // upsert new account
    err = stripedb.UpsertUserAccount(id, "sub1", "plan1", "cust1")
    if err != nil {
        t.Fatalf("UpsertUserAccount failed: %v", err)
    }

    // should exist now
    exists, subID, err := stripedb.CheckUserAccount(id)
    if err != nil || !exists || subID != "sub1" {
        t.Fatalf("CheckUserAccount expected (true, sub1), got (%v, %v), err %v", exists, subID, err)
    }
}
