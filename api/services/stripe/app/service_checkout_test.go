package app

import (
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	stripe "github.com/stripe/stripe-go"
	config "github.com/tbeaudouin05/stripe-trellai/api/config"
	database "github.com/tbeaudouin05/stripe-trellai/api/database"
	stripedb "github.com/tbeaudouin05/stripe-trellai/api/services/stripe/db"
)

const (
	checkoutBoardID     = "checkout-test-board"
	checkoutNoneBoardID = "checkout-non-existent-board"
)

type fakeGateway struct {
	subs  map[string]stripe.Subscription
	custs map[string]stripe.Customer
}

func (f fakeGateway) GetSubscription(id string) (stripe.Subscription, error) {
	if f.subs == nil {
		return stripe.Subscription{}, nil
	}
	return f.subs[id], nil
}

func (f fakeGateway) CancelSubscription(id string) error { return nil }
func (f fakeGateway) GetCustomer(id string) (stripe.Customer, error) {
	if f.custs == nil {
		return stripe.Customer{ID: id}, nil
	}
	return f.custs[id], nil
}

// setupTestDB sets up the test database and returns the DB instance and a cleanup function
func setupTestDB(t *testing.T) (*sql.DB, func()) {
	// Prevent tests from running against production database
	config.CheckNotProdDB()
	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}
	config.AppConfig = cfg
	if err := database.Initialize(); err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	db := database.GetDB()
	// Pre-test cleanup to avoid cross-test contamination
	_, _ = db.Exec("DELETE FROM spending_unit WHERE user_external_id IN ($1, $2)", checkoutBoardID, checkoutNoneBoardID)
	_, _ = db.Exec("DELETE FROM free_credit WHERE user_external_id IN ($1, $2)", checkoutBoardID, checkoutNoneBoardID)
	_, _ = db.Exec("DELETE FROM invalid_subscription WHERE user_external_id IN ($1, $2)", checkoutBoardID, checkoutNoneBoardID)
	_, _ = db.Exec("DELETE FROM user_account WHERE user_external_id IN ($1, $2)", checkoutBoardID, checkoutNoneBoardID)
	cleanup := func() {
		_, _ = db.Exec("DELETE FROM spending_unit WHERE user_external_id IN ($1, $2)", checkoutBoardID, checkoutNoneBoardID)
		_, _ = db.Exec("DELETE FROM free_credit WHERE user_external_id IN ($1, $2)", checkoutBoardID, checkoutNoneBoardID)
		_, _ = db.Exec("DELETE FROM invalid_subscription WHERE user_external_id IN ($1, $2)", checkoutBoardID, checkoutNoneBoardID)
		_, _ = db.Exec("DELETE FROM user_account WHERE user_external_id IN ($1, $2)", checkoutBoardID, checkoutNoneBoardID)
	}
	return db, cleanup
}

func Test_HandleCheckoutSessionCompleted_NoExistingBoard(t *testing.T) {
	_, cleanup := setupTestDB(t)
	defer cleanup()

	// No existing board, gateway won't be called for previous sub
	gw := fakeGateway{}
	svc := NewService(gw)

	session := stripe.CheckoutSession{
		ClientReferenceID: checkoutBoardID,
		Customer:          &stripe.Customer{ID: "cust1"},
		Subscription:      &stripe.Subscription{ID: "sub1"},
	}
	raw, err := json.Marshal(session)
	if err != nil {
		t.Fatalf("failed to marshal session: %v", err)
	}
	evt := stripe.Event{Type: "checkout.session.completed", Data: &stripe.EventData{Raw: raw}}

	err = svc.HandleCheckoutSessionCompleted(evt)
	assert.NoError(t, err)

	account, err := stripedb.GetUserAccount(checkoutBoardID)
	assert.NoError(t, err)
	assert.Equal(t, "sub1", account.StripeSubscriptionID)
	assert.Equal(t, "cust1", account.StripeCustomerID)

	credit, err := stripedb.GetFreeCredit(checkoutBoardID)
	assert.NoError(t, err)
	assert.Equal(t, 5, credit)
}

func Test_HandleCheckoutSessionCompleted_ExistingBoard_CanceledPrevSub(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	if err := stripedb.UpsertUserAccount(checkoutBoardID, "old-sub", "plan-old", "cust-old"); err != nil {
		t.Fatalf("failed to insert existing board: %v", err)
	}

	now := time.Now().Unix()
	gw := fakeGateway{subs: map[string]stripe.Subscription{
		"old-sub": {CancelAt: now - 3600},
	}}
	svc := NewService(gw)

	session := stripe.CheckoutSession{ClientReferenceID: checkoutBoardID, Customer: &stripe.Customer{ID: "cust-new"}, Subscription: &stripe.Subscription{ID: "sub-new"}}
	raw, _ := json.Marshal(session)
	evt := stripe.Event{Type: "checkout.session.completed", Data: &stripe.EventData{Raw: raw}}

	err := svc.HandleCheckoutSessionCompleted(evt)
	assert.NoError(t, err)

	account, err := stripedb.GetUserAccount(checkoutBoardID)
	assert.NoError(t, err)
	assert.Equal(t, "sub-new", account.StripeSubscriptionID)
	assert.Equal(t, "cust-new", account.StripeCustomerID)

	var count int
	err = db.QueryRow("SELECT COUNT(1) FROM invalid_subscription WHERE user_external_id = $1", checkoutBoardID).Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, 0, count)

	credit, err := stripedb.GetFreeCredit(checkoutBoardID)
	assert.NoError(t, err)
	assert.Equal(t, 5, credit)
}

func Test_HandleCheckoutSessionCompleted_ExistingBoard_NonCanceledPrevSub(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	if err := stripedb.UpsertUserAccount(checkoutBoardID, "old-sub", "plan-old", "cust-old"); err != nil {
		t.Fatalf("failed to insert existing board: %v", err)
	}

	gw := fakeGateway{subs: map[string]stripe.Subscription{
		"old-sub": {CancelAt: 0, Status: stripe.SubscriptionStatusActive},
	}}
	svc := NewService(gw)

	session := stripe.CheckoutSession{ClientReferenceID: checkoutBoardID, Customer: &stripe.Customer{ID: "cust-new"}, Subscription: &stripe.Subscription{ID: "sub-new"}}
	raw, _ := json.Marshal(session)
	evt := stripe.Event{Type: "checkout.session.completed", Data: &stripe.EventData{Raw: raw}}

	err := svc.HandleCheckoutSessionCompleted(evt)
	assert.NoError(t, err)

	account, err := stripedb.GetUserAccount(checkoutBoardID)
	assert.NoError(t, err)
	assert.Equal(t, "old-sub", account.StripeSubscriptionID)

	var invalidID string
	err = db.QueryRow("SELECT stripe_subscription_id FROM invalid_subscription WHERE user_external_id = $1", checkoutBoardID).Scan(&invalidID)
	assert.NoError(t, err)
	assert.Equal(t, "sub-new", invalidID)

	credit, err := stripedb.GetFreeCredit(checkoutBoardID)
	assert.NoError(t, err)
	assert.Equal(t, 5, credit)
}
