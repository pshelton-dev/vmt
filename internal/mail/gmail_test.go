package mail

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// stubGoogle stands in for Google's token + Gmail send endpoints and records
// the raw message that was submitted.
func stubGoogle(t *testing.T, tokenBody string, tokenCode int, sendCode int, gotRaw *string) func() {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(tokenCode)
		_, _ = w.Write([]byte(tokenBody))
	})
	mux.HandleFunc("/send", func(w http.ResponseWriter, r *http.Request) {
		if auth := r.Header.Get("Authorization"); auth != "Bearer at-123" {
			t.Errorf("missing/incorrect bearer token: %q", auth)
		}
		var body struct {
			Raw string `json:"raw"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if gotRaw != nil {
			*gotRaw = body.Raw
		}
		w.WriteHeader(sendCode)
		if sendCode/100 != 2 {
			_, _ = w.Write([]byte(`{"error":{"message":"Delegation denied"}}`))
		}
	})
	srv := httptest.NewServer(mux)

	oldT, oldS := GoogleTokenURL, GmailSendURL
	GoogleTokenURL, GmailSendURL = srv.URL+"/token", srv.URL+"/send"
	return func() {
		GoogleTokenURL, GmailSendURL = oldT, oldS
		srv.Close()
	}
}

func testGmail() Gmail {
	return Gmail{ClientID: "cid", ClientSecret: "csec", RefreshToken: "rt", From: "me@gmail.com"}
}

func TestSendGmail(t *testing.T) {
	var raw string
	defer stubGoogle(t, `{"access_token":"at-123"}`, 200, 200, &raw)()

	if err := SendGmail(testGmail(), "you@example.com", "Subj", "Line one\nLine two\n"); err != nil {
		t.Fatalf("SendGmail: %v", err)
	}
	decoded, err := base64.URLEncoding.DecodeString(raw)
	if err != nil {
		t.Fatalf("raw is not base64url: %v", err)
	}
	msg := string(decoded)
	for _, want := range []string{"From: me@gmail.com", "To: you@example.com", "Subject: Subj", "Line one\r\nLine two"} {
		if !strings.Contains(msg, want) {
			t.Errorf("message missing %q:\n%s", want, msg)
		}
	}
}

func TestSendGmailNotConfigured(t *testing.T) {
	if err := SendGmail(Gmail{}, "you@example.com", "s", "b"); err == nil {
		t.Fatal("expected an error when Gmail is not connected")
	}
}

// An expired refresh token (the 7-day "Testing mode" trap) must surface a
// message that points at the fix rather than a bare 400.
func TestSendGmailInvalidGrantIsExplained(t *testing.T) {
	defer stubGoogle(t, `{"error":"invalid_grant","error_description":"Token has been expired or revoked."}`, 400, 200, nil)()

	err := SendGmail(testGmail(), "you@example.com", "s", "b")
	if err == nil {
		t.Fatal("expected an error")
	}
	for _, want := range []string{"invalid_grant", "In production"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error should mention %q, got: %v", want, err)
		}
	}
}

func TestSendGmailAPIErrorSurfaced(t *testing.T) {
	defer stubGoogle(t, `{"access_token":"at-123"}`, 200, 403, nil)()

	err := SendGmail(testGmail(), "you@example.com", "s", "b")
	if err == nil || !strings.Contains(err.Error(), "Delegation denied") {
		t.Fatalf("expected the API message to be surfaced, got: %v", err)
	}
}

func TestExchangeCode(t *testing.T) {
	// id_token payload carrying the account email.
	payload := base64.RawURLEncoding.EncodeToString([]byte(`{"email":"me@gmail.com"}`))
	idTok := "h." + payload + ".sig"
	defer stubGoogle(t, `{"refresh_token":"rt-9","id_token":"`+idTok+`"}`, 200, 200, nil)()

	rt, email, err := ExchangeCode("cid", "csec", "https://x/cb", "code-1")
	if err != nil {
		t.Fatalf("ExchangeCode: %v", err)
	}
	if rt != "rt-9" {
		t.Errorf("refresh token = %q", rt)
	}
	if email != "me@gmail.com" {
		t.Errorf("email = %q", email)
	}
}

// Without a refresh token the connection is useless, so it must be an error
// rather than a silently half-connected account.
func TestExchangeCodeWithoutRefreshToken(t *testing.T) {
	defer stubGoogle(t, `{"id_token":"x.y.z"}`, 200, 200, nil)()

	if _, _, err := ExchangeCode("cid", "csec", "https://x/cb", "code-1"); err == nil {
		t.Fatal("expected an error when Google returns no refresh token")
	}
}

func TestAuthCodeURL(t *testing.T) {
	u := AuthCodeURL("cid", "https://vmt.example/api/v1/oauth/google/callback", "st-1")
	for _, want := range []string{
		"client_id=cid",
		"access_type=offline", // required for a refresh token
		"prompt=consent",      // required to re-issue one
		"state=st-1",
		"gmail.send",
	} {
		if !strings.Contains(u, want) {
			t.Errorf("auth URL missing %q:\n%s", want, u)
		}
	}
}
