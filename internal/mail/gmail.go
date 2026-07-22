package mail

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Google OAuth + Gmail API endpoints. Declared as vars so tests can point them
// at a local stub server.
var (
	GoogleTokenURL = "https://oauth2.googleapis.com/token"
	GoogleAuthURL  = "https://accounts.google.com/o/oauth2/v2/auth"
	GmailSendURL   = "https://gmail.googleapis.com/gmail/v1/users/me/messages/send"
)

// GmailScopes are the permissions requested during the consent flow:
// gmail.send to deliver mail, plus openid/email so we can show which account
// is connected. gmail.send only permits sending — it grants no mailbox read.
var GmailScopes = []string{
	"https://www.googleapis.com/auth/gmail.send",
	"openid",
	"email",
}

const httpTimeout = 20 * time.Second

// Gmail holds the OAuth credentials for sending through a connected Google
// account. RefreshToken is obtained once via the consent flow and then
// exchanged for short-lived access tokens on each send.
type Gmail struct {
	ClientID     string
	ClientSecret string
	RefreshToken string
	From         string // the connected account address (Gmail rejects others)
}

// Configured reports whether a Google account has been connected.
func (g Gmail) Configured() bool {
	return g.ClientID != "" && g.ClientSecret != "" && g.RefreshToken != ""
}

// AuthCodeURL builds the Google consent URL. access_type=offline plus
// prompt=consent is what makes Google return a refresh token — without both,
// re-authorising an already-approved app yields no refresh token at all.
func AuthCodeURL(clientID, redirectURI, state string) string {
	q := url.Values{
		"client_id":     {clientID},
		"redirect_uri":  {redirectURI},
		"response_type": {"code"},
		"scope":         {strings.Join(GmailScopes, " ")},
		"access_type":   {"offline"},
		"prompt":        {"consent"},
		"state":         {state},
	}
	return GoogleAuthURL + "?" + q.Encode()
}

// ExchangeCode trades an authorization code for tokens, returning the refresh
// token and the email address of the account that granted consent.
func ExchangeCode(clientID, clientSecret, redirectURI, code string) (refreshToken, email string, err error) {
	form := url.Values{
		"code":          {code},
		"client_id":     {clientID},
		"client_secret": {clientSecret},
		"redirect_uri":  {redirectURI},
		"grant_type":    {"authorization_code"},
	}
	var out struct {
		RefreshToken string `json:"refresh_token"`
		IDToken      string `json:"id_token"`
	}
	if err := postForm(GoogleTokenURL, form, &out); err != nil {
		return "", "", err
	}
	if out.RefreshToken == "" {
		return "", "", fmt.Errorf("google returned no refresh token (re-consent with prompt=consent)")
	}
	return out.RefreshToken, emailFromIDToken(out.IDToken), nil
}

// accessToken exchanges the stored refresh token for a short-lived access token.
func (g Gmail) accessToken() (string, error) {
	form := url.Values{
		"client_id":     {g.ClientID},
		"client_secret": {g.ClientSecret},
		"refresh_token": {g.RefreshToken},
		"grant_type":    {"refresh_token"},
	}
	var out struct {
		AccessToken string `json:"access_token"`
	}
	if err := postForm(GoogleTokenURL, form, &out); err != nil {
		// The most common cause here is an OAuth app still in "Testing"
		// publishing status, where Google expires refresh tokens after 7 days.
		return "", fmt.Errorf("%w (if this says invalid_grant, reconnect the "+
			"Google account and set the OAuth consent screen to \"In production\")", err)
	}
	if out.AccessToken == "" {
		return "", fmt.Errorf("google returned no access token")
	}
	return out.AccessToken, nil
}

// SendGmail delivers a plaintext message through the Gmail API.
func SendGmail(g Gmail, to, subject, body string) error {
	if !g.Configured() {
		return fmt.Errorf("Gmail is not connected")
	}
	tok, err := g.accessToken()
	if err != nil {
		return err
	}
	raw := base64.URLEncoding.EncodeToString(buildMessage(g.From, to, subject, body))
	payload, _ := json.Marshal(map[string]string{"raw": raw})

	req, err := http.NewRequest("POST", GmailSendURL, strings.NewReader(string(payload)))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")

	res, err := (&http.Client{Timeout: httpTimeout}).Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode/100 != 2 {
		return fmt.Errorf("gmail send failed: %s", apiError(res))
	}
	return nil
}

// postForm posts a URL-encoded form and decodes a JSON response into out.
func postForm(endpoint string, form url.Values, out any) error {
	res, err := (&http.Client{Timeout: httpTimeout}).PostForm(endpoint, form)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode/100 != 2 {
		return fmt.Errorf("google token endpoint: %s", apiError(res))
	}
	return json.NewDecoder(res.Body).Decode(out)
}

// apiError extracts a useful message from a Google error response, falling back
// to the status line when the body isn't the shape we expect.
func apiError(res *http.Response) string {
	b, _ := io.ReadAll(io.LimitReader(res.Body, 4<<10))
	var e struct {
		Error     any    `json:"error"`
		ErrorDesc string `json:"error_description"`
	}
	if json.Unmarshal(b, &e) == nil {
		// OAuth errors: {"error":"invalid_grant","error_description":"..."}
		if s, ok := e.Error.(string); ok && s != "" {
			if e.ErrorDesc != "" {
				return s + ": " + e.ErrorDesc
			}
			return s
		}
		// API errors: {"error":{"message":"..."}}
		var api struct {
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if json.Unmarshal(b, &api) == nil && api.Error.Message != "" {
			return api.Error.Message
		}
	}
	if len(b) > 0 {
		return res.Status + ": " + strings.TrimSpace(string(b))
	}
	return res.Status
}

// emailFromIDToken pulls the "email" claim out of an OIDC id_token. The token
// arrived over TLS directly from Google's token endpoint, so the signature does
// not need re-verifying here — it is only used to label the connected account.
func emailFromIDToken(idToken string) string {
	parts := strings.Split(idToken, ".")
	if len(parts) != 3 {
		return ""
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return ""
	}
	var claims struct {
		Email string `json:"email"`
	}
	if json.Unmarshal(payload, &claims) != nil {
		return ""
	}
	return claims.Email
}
