package validator

import "testing"

type testUserDTO struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
	Role     string `json:"role" validate:"omitempty,oneof=buyer organizer admin"`
}

func TestValidate(t *testing.T) {
	// 1. Test Valid Struct
	validDTO := testUserDTO{
		Email:    "test@example.com",
		Password: "secretpassword",
		Role:     "buyer",
	}

	if errs := Validate(validDTO); errs != nil {
		t.Errorf("expected no validation errors, got: %v", errs)
	}

	// 2. Test Invalid Email and Short Password
	invalidDTO := testUserDTO{
		Email:    "invalid-email",
		Password: "123",
		Role:     "superman",
	}

	errs := Validate(invalidDTO)
	if errs == nil {
		t.Fatalf("expected validation errors, got nil")
	}

	if _, ok := errs["email"]; !ok {
		t.Errorf("expected error for field 'email'")
	}
	if _, ok := errs["password"]; !ok {
		t.Errorf("expected error for field 'password'")
	}
	if _, ok := errs["role"]; !ok {
		t.Errorf("expected error for field 'role'")
	}
}
