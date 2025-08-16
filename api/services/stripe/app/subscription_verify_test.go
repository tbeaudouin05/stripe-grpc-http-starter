package app

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	stripe "github.com/stripe/stripe-go"
	config "github.com/tbeaudouin05/stripe-trellai/api/config"
	database "github.com/tbeaudouin05/stripe-trellai/api/database"
	stripedb "github.com/tbeaudouin05/stripe-trellai/api/services/stripe/db"
)

func hashID(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

const (
	subBoardID       = "sub-test-board"
	subNoneBoardID   = "sub-non-existent-board"
)

// Ensure DB is initialized and perform cleanup for this file's IDs every time.
func setupSubTestDB(t *testing.T) (*sql.DB, func()) {
	if database.GetDB() == nil {
		config.CheckNotProdDB()
		cfg, err := config.LoadConfig()
		if err != nil {
			t.Fatalf("Failed to load config: %v", err)
		}
		config.AppConfig = cfg
		if err := database.Initialize(); err != nil {
			t.Fatalf("Failed to initialize database: %v", err)
		}
	}
	db := database.GetDB()
	// Pre-test cleanup for this file's IDs (hashed)
	hb := hashID(subBoardID)
	hnb := hashID(subNoneBoardID)
	_, _ = db.Exec("DELETE FROM spending_unit WHERE user_external_id IN ($1, $2)", hb, hnb)
	_, _ = db.Exec("DELETE FROM free_credit WHERE user_external_id IN ($1, $2)", hb, hnb)
	_, _ = db.Exec("DELETE FROM invalid_subscription WHERE user_external_id IN ($1, $2)", hb, hnb)
	_, _ = db.Exec("DELETE FROM user_account WHERE user_external_id IN ($1, $2)", hb, hnb)
	cleanup := func() {
		_, _ = db.Exec("DELETE FROM spending_unit WHERE user_external_id IN ($1, $2)", hb, hnb)
		_, _ = db.Exec("DELETE FROM free_credit WHERE user_external_id IN ($1, $2)", hb, hnb)
		_, _ = db.Exec("DELETE FROM invalid_subscription WHERE user_external_id IN ($1, $2)", hb, hnb)
		_, _ = db.Exec("DELETE FROM user_account WHERE user_external_id IN ($1, $2)", hb, hnb)
	}
	return db, cleanup
}

func Test_VerifySubscription_ValidWithFreeCredit(t *testing.T) {
	db, cleanup := setupSubTestDB(t)
	defer cleanup()

	if err := stripedb.UpsertUserAccount(subBoardID, "sub_123", "plan_123", "cust_123"); err != nil {
		t.Fatalf("UpsertUserAccount failed: %v", err)
	}
	if account, err := stripedb.GetUserAccount(subBoardID); err != nil {
		t.Fatalf("GetUserAccount failed: %v", err)
	} else {
		if account.StripeSubscriptionID != "sub_123" || account.StripeCustomerID != "cust_123" {
			t.Fatalf("UserAccount fields not set as expected: got sub=%q cust=%q", account.StripeSubscriptionID, account.StripeCustomerID)
		}
	}
	if _, err := db.Exec("DELETE FROM free_credit WHERE user_external_id = $1", hashID(subBoardID)); err != nil {
		t.Fatalf("Failed to delete free_credit: %v", err)
	}
	if _, err := db.Exec("INSERT INTO free_credit (user_external_id, credit) VALUES ($1, $2)", hashID(subBoardID), 5); err != nil {
		t.Fatalf("Failed to insert free_credit: %v", err)
	}

	svc := NewService(fakeGateway{})
	resp, err := svc.VerifySubscription(subBoardID)
	assert.NoError(t, err)
	assert.True(t, resp.IsValidSubscription)
	assert.Equal(t, ValidityTypeFreeTier, resp.ValidityType)
}

func Test_VerifySubscription_DoesNotExistAndFreeCreditIsNotEnough(t *testing.T) {
	db, cleanup := setupSubTestDB(t)
	defer cleanup()

	if _, err := db.Exec("DELETE FROM spending_unit WHERE user_external_id = $1", hashID(subNoneBoardID)); err != nil {
		t.Fatalf("Failed to delete spending_unit: %v", err)
	}
	if _, err := db.Exec("DELETE FROM user_account WHERE user_external_id = $1", hashID(subNoneBoardID)); err != nil {
		t.Fatalf("Failed to delete user_account: %v", err)
	}
	// With strict FK on free_credit.user_external_id, ensure the account exists first.
	if err := stripedb.UpsertUserAccount(subNoneBoardID, "", "", ""); err != nil {
		t.Fatalf("UpsertUserAccount (for subNoneBoardID) failed: %v", err)
	}
	if _, err := db.Exec("INSERT INTO free_credit (user_external_id, credit) VALUES ($1, 0) ON CONFLICT (user_external_id) DO UPDATE SET credit = 0", hashID(subNoneBoardID)); err != nil {
		t.Fatalf("Failed to upsert free_credit: %v", err)
	}

	svc := NewService(fakeGateway{})
	resp, err := svc.VerifySubscription(subNoneBoardID)
	assert.NoError(t, err)
	assert.False(t, resp.IsValidSubscription)
	assert.Equal(t, InvalidityTypeNoSubscription, resp.InvalidityType)
	assert.Empty(t, resp.StripeCustomerEmail)
}

func Test_VerifySubscription_ValidWithoutFreeCredit(t *testing.T) {
	db, cleanup := setupSubTestDB(t)
	defer cleanup()
	if err := stripedb.UpsertUserAccount(subBoardID, "sub_123", "plan_123", "cust_123"); err != nil {
		t.Fatalf("UpsertUserAccount failed: %v", err)
	}
	if _, err := db.Exec("INSERT INTO free_credit (user_external_id, credit) VALUES ($1, 0) ON CONFLICT (user_external_id) DO UPDATE SET credit = 0", hashID(subBoardID)); err != nil {
		t.Fatalf("Failed to upsert free_credit: %v", err)
	}

	now := time.Now().Unix()
	gw := fakeGateway{
		subs: map[string]stripe.Subscription{
			"sub_123": {CancelAt: 0, Quantity: 1, Status: stripe.SubscriptionStatusActive, Plan: &stripe.Plan{Amount: 1400}, CurrentPeriodStart: now - 86400, CurrentPeriodEnd: now + 86400},
		},
		custs: map[string]stripe.Customer{
			"cust_123": {Email: "valid@example.com"},
		},
	}
	svc := NewService(gw)
	resp, err := svc.VerifySubscription(subBoardID)
	assert.NoError(t, err)
	assert.True(t, resp.IsValidSubscription)
	assert.Empty(t, resp.InvalidityType)
}

func Test_VerifySubscription_Exhausted(t *testing.T) {
	db, cleanup := setupSubTestDB(t)
	defer cleanup()
	if err := stripedb.UpsertUserAccount(subBoardID, "sub_123", "plan_123", "cust_123"); err != nil {
		t.Fatalf("UpsertUserAccount failed: %v", err)
	}
	if _, err := db.Exec("INSERT INTO free_credit (user_external_id, credit) VALUES ($1, 0) ON CONFLICT (user_external_id) DO UPDATE SET credit = 0", hashID(subBoardID)); err != nil {
		t.Fatalf("Failed to upsert free_credit: %v", err)
	}
	if _, err := db.Exec("DELETE FROM spending_unit WHERE user_external_id = $1", hashID(subBoardID)); err != nil {
		t.Fatalf("Failed to delete spending_unit: %v", err)
	}

	currentStart := time.Now().Unix()
	currentEnd := time.Now().Unix() + 86400
	// created_at is stored in ms; convert start to ms for inserted rows
	currentStartMs := currentStart * 1000
	values := make([]string, 45)
	args := make([]interface{}, 0, 45*3)
	for i := 0; i < 45; i++ {
		values[i] = fmt.Sprintf("($%d, $%d, $%d)", i*3+1, i*3+2, i*3+3)
		// keep args order matching column order below (user_external_id, external_id, created_at)
		args = append(args, hashID(subBoardID), fmt.Sprintf("sub-card-%d", i), currentStartMs+int64(i))
	}
	query := "INSERT INTO spending_unit (user_external_id, external_id, created_at) VALUES " + strings.Join(values, ",")
	if _, err := db.Exec(query, args...); err != nil {
		t.Fatalf("Failed to bulk insert spending_unit: %v", err)
	}

	gw := fakeGateway{
		subs: map[string]stripe.Subscription{
			"sub_123": {CancelAt: 0, Status: stripe.SubscriptionStatusActive, Quantity: 1, Plan: &stripe.Plan{Amount: PricePerTicket * 40}, CurrentPeriodStart: currentStart, CurrentPeriodEnd: currentEnd},
		},
		custs: map[string]stripe.Customer{
			"cust_123": {Email: "exhausted@example.com"},
		},
	}
	svc := NewService(gw)
	resp, err := svc.VerifySubscription(subBoardID)
	assert.NoError(t, err)
	assert.False(t, resp.IsValidSubscription)
	assert.Equal(t, InvalidityTypeExhausted, resp.InvalidityType)
	assert.Equal(t, "exhausted@example.com", resp.StripeCustomerEmail)
}

func Test_VerifySubscription_Cancelled(t *testing.T) {
	db, cleanup := setupSubTestDB(t)
	defer cleanup()
	if err := stripedb.UpsertUserAccount(subBoardID, "sub_123", "plan_123", "cust_123"); err != nil {
		t.Fatalf("UpsertUserAccount failed: %v", err)
	}
	if _, err := db.Exec("INSERT INTO free_credit (user_external_id, credit) VALUES ($1, 0) ON CONFLICT (user_external_id) DO UPDATE SET credit = 0", hashID(subBoardID)); err != nil {
		t.Fatalf("Failed to upsert free_credit: %v", err)
	}
	if _, err := db.Exec("DELETE FROM spending_unit WHERE user_external_id = $1", hashID(subBoardID)); err != nil {
		t.Fatalf("Failed to delete spending_unit: %v", err)
	}

	now := time.Now().Unix()
	gw := fakeGateway{
		subs: map[string]stripe.Subscription{
			"sub_123": {CancelAt: now - 3600, CurrentPeriodStart: now - 86400, CurrentPeriodEnd: now + 86400},
		},
		custs: map[string]stripe.Customer{
			"cust_123": {Email: "cancelled@example.com"},
		},
	}
	svc := NewService(gw)
	resp, err := svc.VerifySubscription(subBoardID)
	assert.NoError(t, err)
	assert.False(t, resp.IsValidSubscription)
	assert.Equal(t, InvalidityTypeCancelled, resp.InvalidityType)
	assert.Equal(t, "cancelled@example.com", resp.StripeCustomerEmail)
}
