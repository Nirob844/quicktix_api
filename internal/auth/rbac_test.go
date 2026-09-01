package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRBACAndOwnership(t *testing.T) {
	dummyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	buyerClaims := &JWTClaims{UserID: "usr_buyer_1", Email: "buyer@example.com", Role: RoleBuyer}
	organizerClaims := &JWTClaims{UserID: "usr_org_1", Email: "org@example.com", Role: RoleOrganizer}
	adminClaims := &JWTClaims{UserID: "usr_admin_1", Email: "admin@example.com", Role: RoleAdmin}

	// 1. Test RequireBuyer
	buyerHandler := RequireBuyer()(dummyHandler)

	// Buyer context -> 200 OK
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/reserve", nil).WithContext(WithUserContext(context.Background(), buyerClaims))
	buyerHandler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 OK for buyer in RequireBuyer, got %d", rec.Code)
	}

	// Organizer context -> 403 Forbidden
	recOrg := httptest.NewRecorder()
	reqOrg := httptest.NewRequest("GET", "/reserve", nil).WithContext(WithUserContext(context.Background(), organizerClaims))
	buyerHandler.ServeHTTP(recOrg, reqOrg)
	if recOrg.Code != http.StatusForbidden {
		t.Errorf("expected 403 Forbidden for organizer in RequireBuyer, got %d", recOrg.Code)
	}

	// 2. Test RequireOrganizer
	orgHandler := RequireOrganizer()(dummyHandler)

	// Organizer -> 200 OK
	recOrgAccess := httptest.NewRecorder()
	orgHandler.ServeHTTP(recOrgAccess, reqOrg)
	if recOrgAccess.Code != http.StatusOK {
		t.Errorf("expected 200 OK for organizer in RequireOrganizer, got %d", recOrgAccess.Code)
	}

	// Admin -> 200 OK
	recAdminAccess := httptest.NewRecorder()
	reqAdmin := httptest.NewRequest("GET", "/events/create", nil).WithContext(WithUserContext(context.Background(), adminClaims))
	orgHandler.ServeHTTP(recAdminAccess, reqAdmin)
	if recAdminAccess.Code != http.StatusOK {
		t.Errorf("expected 200 OK for admin in RequireOrganizer, got %d", recAdminAccess.Code)
	}

	// Buyer -> 403 Forbidden
	recBuyerOrg := httptest.NewRecorder()
	orgHandler.ServeHTTP(recBuyerOrg, req)
	if recBuyerOrg.Code != http.StatusForbidden {
		t.Errorf("expected 403 Forbidden for buyer in RequireOrganizer, got %d", recBuyerOrg.Code)
	}

	// 3. Test RequireAdmin
	adminHandler := RequireAdmin()(dummyHandler)

	// Admin -> 200 OK
	recAdminPass := httptest.NewRecorder()
	adminHandler.ServeHTTP(recAdminPass, reqAdmin)
	if recAdminPass.Code != http.StatusOK {
		t.Errorf("expected 200 OK for admin in RequireAdmin, got %d", recAdminPass.Code)
	}

	// Organizer -> 403 Forbidden
	recOrgFail := httptest.NewRecorder()
	adminHandler.ServeHTTP(recOrgFail, reqOrg)
	if recOrgFail.Code != http.StatusForbidden {
		t.Errorf("expected 403 Forbidden for organizer in RequireAdmin, got %d", recOrgFail.Code)
	}

	// 4. Test CheckOwnership
	// Owner matches resource -> true
	ctxOrg := WithUserContext(context.Background(), organizerClaims)
	if !CheckOwnership(ctxOrg, "usr_org_1") {
		t.Errorf("expected CheckOwnership to be true when user is resource owner")
	}

	// Non-owner organizer -> false
	if CheckOwnership(ctxOrg, "usr_org_other") {
		t.Errorf("expected CheckOwnership to be false for non-owner organizer")
	}

	// Admin accesses another user's resource -> true (Admin bypass)
	ctxAdmin := WithUserContext(context.Background(), adminClaims)
	if !CheckOwnership(ctxAdmin, "usr_org_1") {
		t.Errorf("expected CheckOwnership to be true for Admin on any resource")
	}
}
