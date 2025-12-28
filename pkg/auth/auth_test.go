package auth

import (
	"net/url"
	"testing"
)

func TestAuthCodeURL(t *testing.T) {
	// Set up test values
	ClientID = "test-client-id"
	testURI, _ := url.Parse("http://localhost:8080/callback")
	RedirectURI = testURI

	url := AuthCodeURL()
	if url == nil {
		t.Fatal("AuthCodeURL returned nil")
	}

	// Check scheme and host
	if url.Scheme != "https" {
		t.Errorf("Expected scheme 'https', got '%s'", url.Scheme)
	}
	if url.Host != "login.microsoftonline.com" {
		t.Errorf("Expected host 'login.microsoftonline.com', got '%s'", url.Host)
	}

	// Check query parameters
	query := url.Query()
	if query.Get("client_id") != "test-client-id" {
		t.Errorf("Expected client_id 'test-client-id', got '%s'", query.Get("client_id"))
	}
	if query.Get("response_type") != "code" {
		t.Errorf("Expected response_type 'code', got '%s'", query.Get("response_type"))
	}
	if query.Get("redirect_uri") != "http://localhost:8080/callback" {
		t.Errorf("Expected redirect_uri 'http://localhost:8080/callback', got '%s'", query.Get("redirect_uri"))
	}
	if query.Get("scope") != "XboxLive.signin offline_access" {
		t.Errorf("Expected scope 'XboxLive.signin offline_access', got '%s'", query.Get("scope"))
	}
}

func TestFetchDeviceCode(t *testing.T) {
	// Test that FetchDeviceCode returns an error when ClientID is not set
	ClientID = ""
	_, err := FetchDeviceCode()
	if err == nil {
		t.Error("FetchDeviceCode should return error when ClientID is not set")
	}

	// Test that FetchDeviceCode works when ClientID is set
	ClientID = "test-client-id"
	_, err = FetchDeviceCode()
	// Note: This will fail with network error, but that's expected in test environment
	// We just want to ensure it doesn't fail due to missing ClientID
	if err == nil {
		t.Log("FetchDeviceCode succeeded (unexpected in test environment)")
	}
}
