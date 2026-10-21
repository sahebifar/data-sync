package models

import (
	"testing"
	"time"
)

func TestUserFields(t *testing.T) {
	now := time.Date(2026, time.October, 20, 12, 0, 0, 0, time.UTC)

	user := User{
		ID:        1,
		FirstName: "Amir",
		LastName:  "Sahebifar",
		Email:     "amir@test.com",
		Age:       26,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if user.ID != 1 {
		t.Errorf("expected ID 1, got %d", user.ID)
	}
	if user.FirstName != "Amir" {
		t.Errorf("expected FirstName Amir, got %s", user.FirstName)
	}
	if user.LastName != "Sahebifar" {
		t.Errorf("expected LastName Sahebifar, got %s", user.LastName)
	}
	if user.Email != "amir@test.com" {
		t.Errorf("expected Email amir@test.com, got %s", user.Email)
	}
	if user.Age != 26 {
		t.Errorf("expected Age 26, got %d", user.Age)
	}
	if !user.CreatedAt.Equal(now) {
		t.Error("CreatedAt does not match expected time")
	}
	if !user.UpdatedAt.Equal(now) {
		t.Error("UpdatedAt does not match expected time")
	}
}