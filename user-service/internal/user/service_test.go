package user

import (
	"context"
	"strings"
	"testing"
	"time"
)

const testSecret = "my-test-jwt-secret-key-12345"

func TestRegister_Success(t *testing.T) {
	repo := newMockRepository()
	emailSvc := newMockEmailService()
	eventSvc := newMockEventService()

	svc := NewService(repo, testSecret, emailSvc, eventSvc, nil, nil)

	input := &RegisterInput{
		Email:     "yerlan@kazakhexpress.kz",
		Password:  "SecurePassword123!",
		FirstName: "Yerlan",
		LastName:  "Almaty",
		Phone:     "+77011112233",
		Address:   "Dostyk Ave 45",
	}

	resp, err := svc.Register(input)
	if err != nil {
		t.Fatalf("Register() failed: %v", err)
	}

	if resp.User.Email != input.Email {
		t.Errorf("expected email %q, got %q", input.Email, resp.User.Email)
	}
	if resp.User.Password != "" {
		t.Error("expected password to be cleared in response")
	}
	if resp.Token == "" {
		t.Error("expected access token to be generated")
	}
	if resp.RefreshToken == "" {
		t.Error("expected refresh token to be generated")
	}

	// Verify welcome email trigger
	time.Sleep(10 * time.Millisecond) // wait for goroutines
	emailSvc.mu.Lock()
	firstName, emailSent := emailSvc.welcomeSent[input.Email]
	emailSvc.mu.Unlock()
	if !emailSent {
		t.Error("welcome email was not sent")
	}
	if firstName != input.FirstName {
		t.Errorf("expected welcome email first name %q, got %q", input.FirstName, firstName)
	}

	// Verify event published
	eventSvc.mu.Lock()
	publishedCount := len(eventSvc.events)
	eventSvc.mu.Unlock()
	if publishedCount != 1 {
		t.Errorf("expected 1 event published, got %d", publishedCount)
	}
}

func TestRegister_DuplicateEmail(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo, testSecret, nil, nil, nil, nil)

	input := &RegisterInput{
		Email:     "duplicate@kazakhexpress.kz",
		Password:  "pass123",
		FirstName: "First",
		LastName:  "Last",
	}

	_, err := svc.Register(input)
	if err != nil {
		t.Fatalf("first registration failed: %v", err)
	}

	_, err = svc.Register(input)
	if err == nil {
		t.Error("expected error when registering with a duplicate email")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestLogin_Success(t *testing.T) {
	repo := newMockRepository()
	rateLimit := newMockRateLimitService()
	svc := NewService(repo, testSecret, nil, nil, nil, rateLimit)

	// Register user first
	regInput := &RegisterInput{
		Email:    "login-test@kazakhexpress.kz",
		Password: "SecretPassword",
	}
	_, err := svc.Register(regInput)
	if err != nil {
		t.Fatalf("registration failed: %v", err)
	}

	// Reset attempts for login testing
	rateLimit.mu.Lock()
	delete(rateLimit.attempts, regInput.Email)
	rateLimit.mu.Unlock()

	loginInput := &LoginInput{
		Email:    regInput.Email,
		Password: regInput.Password,
	}

	resp, err := svc.Login(loginInput)
	if err != nil {
		t.Fatalf("Login() failed: %v", err)
	}

	if resp.User.Email != loginInput.Email {
		t.Errorf("expected email %q, got %q", loginInput.Email, resp.User.Email)
	}
	if resp.Token == "" || resp.RefreshToken == "" {
		t.Error("expected token pair to be returned")
	}

	// Verify rate limit check was made and then reset
	rateLimit.mu.Lock()
	attempts := rateLimit.attempts[regInput.Email]
	rateLimit.mu.Unlock()
	if attempts != 0 {
		t.Errorf("expected rate limit to be reset, but attempts is %d", attempts)
	}
}

func TestLogin_InvalidPassword(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo, testSecret, nil, nil, nil, nil)

	regInput := &RegisterInput{
		Email:    "login-fail@kazakhexpress.kz",
		Password: "RealPassword",
	}
	_, _ = svc.Register(regInput)

	loginInput := &LoginInput{
		Email:    regInput.Email,
		Password: "WrongPassword",
	}

	_, err := svc.Login(loginInput)
	if err == nil {
		t.Error("expected error for invalid password")
	}
	if !strings.Contains(err.Error(), "invalid email or password") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestGetProfile_Caching(t *testing.T) {
	repo := newMockRepository()
	cache := newMockCacheService()
	svc := NewService(repo, testSecret, nil, nil, cache, nil)

	// Create user
	regInput := &RegisterInput{
		Email:     "profile-cached@kazakhexpress.kz",
		Password:  "password",
		FirstName: "Bauyrzhan",
	}
	regResp, err := svc.Register(regInput)
	if err != nil {
		t.Fatalf("registration failed: %v", err)
	}
	userID := regResp.User.ID

	// Verify Cache Miss -> Loads from database -> Caches
	profile1, err := svc.GetProfile(userID)
	if err != nil {
		t.Fatalf("GetProfile() failed on first call: %v", err)
	}
	if profile1.FirstName != "Bauyrzhan" {
		t.Errorf("expected name Bauyrzhan, got %s", profile1.FirstName)
	}

	// Wait briefly for the async caching goroutine
	time.Sleep(10 * time.Millisecond)

	cache.mu.Lock()
	_, inCache := cache.users[userID]
	cache.mu.Unlock()
	if !inCache {
		t.Error("user was not cached after database load")
	}

	// Modify in database directly to prove next call hits cache
	repo.mu.Lock()
	repo.users[userID].FirstName = "Modified"
	repo.mu.Unlock()

	profile2, err := svc.GetProfile(userID)
	if err != nil {
		t.Fatalf("GetProfile() failed on second call: %v", err)
	}
	if profile2.FirstName != "Bauyrzhan" {
		t.Errorf("expected cached name Bauyrzhan, but got %s (indicates database was hit directly)", profile2.FirstName)
	}
}

func TestUpdateProfile_InvalidatesCacheAndPublishes(t *testing.T) {
	repo := newMockRepository()
	cache := newMockCacheService()
	eventSvc := newMockEventService()
	svc := NewService(repo, testSecret, nil, eventSvc, cache, nil)

	regResp, _ := svc.Register(&RegisterInput{
		Email:     "update@kazakhexpress.kz",
		Password:  "password",
		FirstName: "OldName",
	})
	userID := regResp.User.ID

	// Seed cache
	_ = cache.CacheUser(context.Background(), userID, &regResp.User, time.Hour)

	newName := "NewName"
	updateInput := &UpdateProfileInput{
		FirstName: &newName,
	}

	updatedUser, err := svc.UpdateProfile(userID, updateInput)
	if err != nil {
		t.Fatalf("UpdateProfile() failed: %v", err)
	}

	if updatedUser.FirstName != "NewName" {
		t.Errorf("expected name to be NewName, got %s", updatedUser.FirstName)
	}

	// Wait briefly for background goroutines
	time.Sleep(10 * time.Millisecond)

	// Check cache invalidated
	cache.mu.Lock()
	_, inCache := cache.users[userID]
	invalidated := len(cache.invalidated) > 0 && cache.invalidated[0] == userID
	cache.mu.Unlock()
	if inCache || !invalidated {
		t.Error("expected cache to be invalidated")
	}

	// Check event published
	eventSvc.mu.Lock()
	publishedCount := len(eventSvc.events)
	eventSvc.mu.Unlock()
	if publishedCount != 2 { // 1 for registration, 1 for update
		t.Errorf("expected 2 events published in total, got %d", publishedCount)
	}
}

func TestValidateToken_BlacklistAndValidity(t *testing.T) {
	repo := newMockRepository()
	cache := newMockCacheService()
	svc := NewService(repo, testSecret, nil, nil, cache, nil)

	regResp, _ := svc.Register(&RegisterInput{
		Email:    "token-test@kazakhexpress.kz",
		Password: "password",
	})

	// 1. Validate proper token
	userID, err := svc.ValidateToken(regResp.Token)
	if err != nil {
		t.Fatalf("ValidateToken() failed for valid token: %v", err)
	}
	if userID != regResp.User.ID {
		t.Errorf("expected user ID %q, got %q", regResp.User.ID, userID)
	}

	// 2. Validate blacklisted token
	_ = cache.BlacklistToken(context.Background(), regResp.Token, time.Hour)
	_, err = svc.ValidateToken(regResp.Token)
	if err == nil {
		t.Error("expected validation to fail for blacklisted token")
	}
	if !strings.Contains(err.Error(), "revoked") {
		t.Errorf("unexpected error: %v", err)
	}

	// 3. Validate corrupt token
	_, err = svc.ValidateToken("corrupt-token-string")
	if err == nil {
		t.Error("expected validation to fail for corrupt token")
	}
}

func TestForgotPassword_And_ResetPassword(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo, testSecret, nil, nil, nil, nil)

	email := "forgot-pwd@kazakhexpress.kz"
	regResp, _ := svc.Register(&RegisterInput{
		Email:    email,
		Password: "OldSecurePassword",
	})
	userID := regResp.User.ID

	// 1. ForgotPassword
	err := svc.ForgotPassword(&ForgotPasswordInput{Email: email})
	if err != nil {
		t.Fatalf("ForgotPassword() failed: %v", err)
	}

	// Find the token that was saved
	repo.mu.Lock()
	var resetHash string
	for uid, info := range repo.resetTokens {
		if uid == userID {
			resetHash = info.tokenHash
		}
	}
	repo.mu.Unlock()

	if resetHash == "" {
		t.Fatal("expected reset token hash to be saved in mock repo")
	}

	// Simulate getting the raw token from the log or mock.
	// Since we hash it internally in forgot password, we can stub a raw reset token that resolves to the same hash.
	// In the real flow, the raw token is printed in logs/sent in email, and reset password takes the raw token, hashes it, and matches it.
	// In our mock, let's reset the mock token to a known raw token so we can test ResetPassword.
	rawToken := "super-secret-random-reset-token-guid"
	knownHash := hashToken(rawToken)
	_ = repo.SavePasswordResetToken(userID, knownHash, time.Now().Add(time.Hour))

	// 2. ResetPassword
	resetInput := &ResetPasswordInput{
		Token:       rawToken,
		NewPassword: "BrandNewSecurePassword123!",
	}
	err = svc.ResetPassword(resetInput)
	if err != nil {
		t.Fatalf("ResetPassword() failed: %v", err)
	}

	// 3. Verify Login with New Password
	loginResp, err := svc.Login(&LoginInput{
		Email:    email,
		Password: "BrandNewSecurePassword123!",
	})
	if err != nil {
		t.Fatalf("Login failed with new password: %v", err)
	}
	if loginResp.User.ID != userID {
		t.Error("failed to log in after password reset")
	}

	// 4. Verify old password no longer works
	_, err = svc.Login(&LoginInput{
		Email:    email,
		Password: "OldSecurePassword",
	})
	if err == nil {
		t.Error("expected login to fail with old password after reset")
	}
}

func TestRefreshToken_SuccessAndExpiration(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo, testSecret, nil, nil, nil, nil)

	regResp, err := svc.Register(&RegisterInput{
		Email:    "refresh-test@kazakhexpress.kz",
		Password: "password",
	})
	if err != nil {
		t.Fatalf("Register() failed: %v", err)
	}

	// 1. Successful Refresh
	refreshResp, err := svc.RefreshToken(regResp.RefreshToken)
	if err != nil {
		t.Fatalf("RefreshToken() failed: %v", err)
	}
	if refreshResp.Token == "" || refreshResp.RefreshToken == "" {
		t.Error("expected new token pair")
	}

	// 2. Refresh with old refresh token should fail (as it was deleted during refresh)
	_, err = svc.RefreshToken(regResp.RefreshToken)
	if err == nil {
		t.Error("expected refresh token to fail with already used token")
	}

	// 3. Refresh with expired token should fail
	newRegResp, _ := svc.Register(&RegisterInput{
		Email:    "refresh-expire@kazakhexpress.kz",
		Password: "password",
	})
	// Force expire in mock
	repo.mu.Lock()
	for _, tok := range repo.tokens {
		if tok.UserID == newRegResp.User.ID {
			tok.ExpiresAt = time.Now().Add(-time.Hour)
		}
	}
	repo.mu.Unlock()

	_, err = svc.RefreshToken(newRegResp.RefreshToken)
	if err == nil {
		t.Error("expected RefreshToken to fail with expired token")
	}
}

func TestLogout(t *testing.T) {
	repo := newMockRepository()
	cache := newMockCacheService()
	svc := NewService(repo, testSecret, nil, nil, cache, nil)

	regResp, _ := svc.Register(&RegisterInput{
		Email:    "logout-test@kazakhexpress.kz",
		Password: "password",
	})

	_ = cache.CacheUser(context.Background(), regResp.User.ID, &regResp.User, time.Hour)

	err := svc.Logout(regResp.User.ID, regResp.Token, regResp.RefreshToken)
	if err != nil {
		t.Fatalf("Logout() failed: %v", err)
	}

	time.Sleep(10 * time.Millisecond) // wait for goroutines

	// Verify access token was blacklisted
	cache.mu.Lock()
	_, blacklisted := cache.blacklist[regResp.Token]
	cache.mu.Unlock()
	if !blacklisted {
		t.Error("expected access token to be blacklisted on logout")
	}

	// Verify refresh tokens were deleted from database
	repo.mu.Lock()
	tokenCount := len(repo.tokens)
	repo.mu.Unlock()
	if tokenCount != 0 {
		t.Errorf("expected refresh tokens to be deleted from repo, but got %d remaining", tokenCount)
	}
}
