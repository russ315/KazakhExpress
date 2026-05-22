//go:build integration

package user

import (
	"os"
	"testing"
	"time"
)

func TestPostgresRepositoryUserLifecycle(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL is required for integration tests")
	}
	repo, err := NewPostgresRepository(dsn)
	if err != nil {
		t.Fatalf("NewPostgresRepository() error = %v", err)
	}
	t.Cleanup(func() { _ = repo.Close() })

	id := "it-user-" + time.Now().Format("150405.000000000")
	user := &User{
		ID: id, Email: id + "@example.com", Password: "hash",
		FirstName: "Integration", LastName: "User", Phone: "+77000000000", Address: "Almaty",
	}
	if err := repo.Create(user); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	found, err := repo.GetByEmail(user.Email)
	if err != nil {
		t.Fatalf("GetByEmail() error = %v", err)
	}
	if found.ID != id {
		t.Fatalf("ID = %q, want %q", found.ID, id)
	}
	found.FirstName = "Updated"
	if err := repo.Update(found); err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if err := repo.AddToBlacklist("jti-"+id, time.Now().Add(time.Hour)); err != nil {
		t.Fatalf("AddToBlacklist() error = %v", err)
	}
	blacklisted, err := repo.IsBlacklisted("jti-" + id)
	if err != nil {
		t.Fatalf("IsBlacklisted() error = %v", err)
	}
	if !blacklisted {
		t.Fatal("token is not blacklisted")
	}
}
