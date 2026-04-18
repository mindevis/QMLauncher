// Package auth provides functions related to game authentication.
package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"QMLauncher/internal/network"
)

// authHTTP is used for Microsoft/Xbox/Minecraft Services (debug-traced when launcher_debug is on; OAuth bodies omitted in logs).
var authHTTP = network.HTTPClientMetadata

const scope = "XboxLive.signin offline_access"

// DefaultRedirectURI is the OAuth redirect URI; must match the URI registered in Azure AD.
const DefaultRedirectURI = "http://localhost:8000/signin"

var ClientID string // Client ID for the Azure application; must be set (QMAdmin or QMLAUNCHER_MSA_CLIENT_ID)

func init() {
	if id := os.Getenv("QMLAUNCHER_MSA_CLIENT_ID"); id != "" {
		ClientID = strings.TrimSpace(id)
	}
}

// IsMicrosoftAuthConfigured returns true if MSA Client ID is set (required for Microsoft/Mojang login).
func IsMicrosoftAuthConfigured() bool {
	return strings.TrimSpace(ClientID) != ""
}

var RedirectURI *url.URL // Redirect URI for the OAuth2 authorization code grant

// A Session holds the necessary information to start Minecraft authenticated.
type Session struct {
	UUID        string
	Username    string
	AccessToken string
}

// AuthCodeURL returns an authorization code URL for the user to navigate to
//
// Used for the OAuth2 authorization code grant
func AuthCodeURL() *url.URL {
	query := url.Values{
		"client_id":     {ClientID},
		"response_type": {"code"},
		"redirect_uri":  {RedirectURI.String()},
		"scope":         {scope},
		"response_mode": {"query"},
	}
	uri, _ := url.Parse("https://login.microsoftonline.com/consumers/oauth2/v2.0/authorize")
	uri.RawQuery = query.Encode()
	return uri
}

// A deviceCodeResponse contains information about device codes to be entered by the user to complete authentication, when they expire, and how often they should be polled for.
type deviceCodeResponse struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
	Message         string `json:"message"`
}

// FetchDeviceCode returns a device code for the user to input to authenticate
//
// Used for the OAuth2 device code grant
func FetchDeviceCode() (deviceCodeResponse, error) {
	params := url.Values{
		"client_id": {ClientID},
		"scope":     {scope},
	}
	resp, err := authHTTP.Post("https://login.microsoftonline.com/consumers/oauth2/v2.0/devicecode", "application/x-www-form-urlencoded", strings.NewReader(params.Encode()))
	if err != nil {
		return deviceCodeResponse{}, err
	}
	defer resp.Body.Close()
	if err := network.CheckResponse(resp); err != nil {
		return deviceCodeResponse{}, err
	}
	var data deviceCodeResponse
	body, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(body, &data); err != nil {
		return deviceCodeResponse{}, err
	}

	return data, nil
}

type msaResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	Scope        string `json:"scope"`
	IDToken      string `json:"id_token"`
	// error response

	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

func authenticateMSA(payload url.Values) (msaResponse, error) {
	var data msaResponse
	resp, err := authHTTP.Post("https://login.microsoftonline.com/consumers/oauth2/v2.0/token", "application/x-www-form-urlencoded", strings.NewReader(payload.Encode()))
	if err != nil {
		return msaResponse{}, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(body, &data); err != nil {
		return msaResponse{}, err
	}
	return data, nil
}

type xblResponse struct {
	Token         string `json:"Token"`
	DisplayClaims struct {
		Xui []struct {
			Uhs string `string:"uhs"`
		} `json:"xui"`
	} `json:"DisplayClaims"`
	IssueInstant time.Time `json:"IssueInstant"`
	NotAfter     time.Time `json:"NotAfter"`
}

func authenticateXBL(msaAccessToken string) (xblResponse, error) {
	type properties struct {
		AuthMethod string `json:"AuthMethod"`
		SiteName   string `json:"SiteName"`
		RpsTicket  string `json:"RpsTicket"`
	}
	type request struct {
		Properties   properties `json:"Properties"`
		TokenType    string     `json:"TokenType"`
		RelyingParty string     `json:"RelyingParty"`
	}
	req, _ := json.Marshal(
		request{
			Properties: properties{
				AuthMethod: "RPS",
				SiteName:   "user.auth.xboxlive.com",
				RpsTicket:  "d=" + msaAccessToken,
			},
			TokenType:    "JWT",
			RelyingParty: "http://auth.xboxlive.com",
		})
	resp, err := authHTTP.Post("https://user.auth.xboxlive.com/user/authenticate", "application/json", strings.NewReader(string(req)))
	if err != nil {
		return xblResponse{}, err
	}
	if err := network.CheckResponse(resp); err != nil {
		return xblResponse{}, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var data xblResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return xblResponse{}, err
	}
	return data, nil
}

type xstsResponse struct {
	Token        string    `json:"Token"`
	IssueInstant time.Time `json:"IssueInstant"`
	NotAfter     time.Time `json:"NotAfter"`
	// error response

	XErr int `json:"XErr"`
}

func authenticateXSTS(xblToken string) (xstsResponse, error) {
	type properties struct {
		SandboxID  string   `json:"SandboxId"`
		UserTokens []string `json:"UserTokens"`
	}
	type request struct {
		Properties   properties `json:"Properties"`
		RelyingParty string     `json:"RelyingParty"`
		TokenType    string     `json:"TokenType"`
	}

	req, _ := json.Marshal(request{
		Properties: properties{
			SandboxID:  "RETAIL",
			UserTokens: []string{xblToken},
		},
		RelyingParty: "rp://api.minecraftservices.com/",
		TokenType:    "JWT",
	})
	resp, err := authHTTP.Post("https://xsts.auth.xboxlive.com/xsts/authorize", "application/json", strings.NewReader(string(req)))
	if err != nil {
		return xstsResponse{}, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var data xstsResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return xstsResponse{}, err
	}

	if err := network.CheckResponse(resp); err != nil {
		if data.XErr != 0 {
			return xstsResponse{}, fmt.Errorf("got error %d", data.XErr)
		}
		return xstsResponse{}, err
	}
	return data, nil
}

type minecraftResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}
type minecraftProfile struct {
	Name string `json:"name"`
	ID   string `json:"id"`
	// error response

	Error        string `json:"error"`
	ErrorMessage string `json:"errorMessage"`
}

func authenticateMinecraft(xstsToken string, userhash string) (minecraftResponse, minecraftProfile, error) {
	type request struct {
		IdentityToken string `json:"identityToken"`
	}

	reqBody, err := json.Marshal(request{
		IdentityToken: fmt.Sprintf("XBL3.0 x=%s;%s", userhash, xstsToken),
	})
	if err != nil {
		return minecraftResponse{}, minecraftProfile{}, err
	}
	req, err := http.NewRequest("POST", "https://api.minecraftservices.com/authentication/login_with_xbox", bytes.NewReader(reqBody))
	if err != nil {
		return minecraftResponse{}, minecraftProfile{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	resp, err := authHTTP.Do(req)
	if err != nil {
		return minecraftResponse{}, minecraftProfile{}, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return minecraftResponse{}, minecraftProfile{}, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		type mcSvcErr struct {
			Path         string `json:"path"`
			ErrorMessage string `json:"errorMessage"`
			Error        string `json:"error"`
			Details      struct {
				Reason string `json:"reason"`
			} `json:"details"`
		}
		var mce mcSvcErr
		_ = json.Unmarshal(body, &mce)
		msg := strings.TrimSpace(mce.ErrorMessage)
		if msg == "" {
			msg = strings.TrimSpace(mce.Error)
		}
		if msg == "" && mce.Details.Reason != "" {
			msg = mce.Details.Reason
		}
		if msg == "" {
			msg = strings.TrimSpace(string(body))
		}
		if msg == "" {
			return minecraftResponse{}, minecraftProfile{}, fmt.Errorf("POST https://api.minecraftservices.com/authentication/login_with_xbox (%d)", resp.StatusCode)
		}
		if resp.StatusCode == http.StatusForbidden && (strings.Contains(strings.ToLower(msg), "app registration") || strings.Contains(strings.ToLower(msg), "invalid app")) {
			msg += " — if this is your own Azure app, request Minecraft API access: https://aka.ms/MinecraftAppReg"
		}
		return minecraftResponse{}, minecraftProfile{}, fmt.Errorf("%s", msg)
	}
	var data minecraftResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return minecraftResponse{}, minecraftProfile{}, err
	}

	profReq, _ := http.NewRequest("GET", "https://api.minecraftservices.com/minecraft/profile", nil)
	profReq.Header.Add("Authorization", "Bearer "+data.AccessToken)
	profReq.Header.Set("Accept", "application/json")
	profResp, err := authHTTP.Do(profReq)
	if err != nil {
		return minecraftResponse{}, minecraftProfile{}, err
	}
	defer profResp.Body.Close()
	profBody, err := io.ReadAll(profResp.Body)
	if err != nil {
		return minecraftResponse{}, minecraftProfile{}, err
	}
	var profile minecraftProfile
	if err := json.Unmarshal(profBody, &profile); err != nil {
		return minecraftResponse{}, minecraftProfile{}, err
	}
	if err := network.CheckResponse(profResp); err != nil {
		if profile.ErrorMessage != "" {
			return minecraftResponse{}, minecraftProfile{}, errors.New(profile.ErrorMessage)
		}
		if profile.Error != "" {
			return minecraftResponse{}, minecraftProfile{}, errors.New(profile.Error)
		}
		return minecraftResponse{}, minecraftProfile{}, err
	}
	return data, profile, nil
}

var ErrNoAccount = errors.New("no account found")

// Authenticate authenticates with all necessary endpoints, or cached data if available and returns a Session.
func Authenticate() (Session, error) {
	if Store.MSA.RefreshToken == "" {
		return Session{}, ErrNoAccount
	}
	if !Store.MSA.isValid() {
		if err := Store.MSA.refresh(); err != nil {
			return Session{}, fmt.Errorf("authenticate with MSA: %w", err)
		}
	}
	if !Store.XBL.isValid() {
		if err := Store.XBL.refresh(); err != nil {
			return Session{}, fmt.Errorf("authenticate with Xbox Live: %w", err)
		}
	}
	if !Store.XSTS.isValid() {
		if err := Store.XSTS.refresh(); err != nil {
			return Session{}, fmt.Errorf("authenticate with XSTS: %w", err)
		}
	}
	if !Store.Minecraft.isValid() {
		if err := Store.Minecraft.refresh(); err != nil {
			return Session{}, fmt.Errorf("authenticate with Minecraft: %w", err)
		}
	}
	if err := Store.WriteToCache(); err != nil {
		return Session{}, fmt.Errorf("write auth store: %w", err)
	}
	return Session{
		Username:    Store.Minecraft.Username,
		UUID:        Store.Minecraft.UUID,
		AccessToken: Store.Minecraft.AccessToken,
	}, nil
}

// AuthenticateWithRedirect authenticates using the OAuth2 Code flow.
//
// success is a string to be shown to the user upon successful authentication.
// fail is shown if an authentication error occurs.
//
// This function blocks until a response has been received on the local authentication server.
func AuthenticateWithRedirect(success, fail string) (Session, error) {
	var code string
	var err error

	port := RedirectURI.Port()
	if port == "" {
		return Session{}, fmt.Errorf("redirect URL must have port specified")
	}
	server := &http.Server{Addr: ":" + port, Handler: nil}
	http.HandleFunc(RedirectURI.Path, func(w http.ResponseWriter, req *http.Request) {
		params := req.URL.Query()
		if params.Get("error") != "" {
			fmt.Fprint(w, fail+"\n"+params.Get("error_description"))
			err = fmt.Errorf("got error: %s", params.Get("error_description"))
		} else {
			fmt.Fprint(w, success)
		}
		code = params.Get("code")
		go server.Shutdown(context.Background())
	})
	server.ListenAndServe()
	if err != nil {
		return Session{}, err
	}

	resp, err := authenticateMSA(url.Values{
		"client_id":    {ClientID},
		"scope":        {scope},
		"redirect_uri": {RedirectURI.String()},
		"grant_type":   {"authorization_code"},
		"code":         {code},
	})
	if err != nil {
		return Session{}, fmt.Errorf("authenticate with MSA: %w", err)
	}
	Store.MSA.write(resp)

	return Authenticate()
}

// AuthenticateWithCode authenticates with a device code.
//
// This function blocks until the user has been authenticated, or another error has occurred.
func AuthenticateWithCode(codeResp deviceCodeResponse) (Session, error) {
	for {
		resp, err := authenticateMSA(url.Values{
			"grant_type":  {"urn:ietf:params:oauth:grant-type:device_code"},
			"client_id":   {ClientID},
			"device_code": {codeResp.DeviceCode},
		})
		if err != nil {
			return Session{}, fmt.Errorf("authenticate with MSA: %w", err)
		}

		switch resp.Error {
		case "authorization_pending":
			time.Sleep(time.Second * time.Duration(codeResp.Interval))
			continue
		case "authorization_declined":
			return Session{}, fmt.Errorf("authorization was declined")
		case "":
			Store.MSA.write(resp)
		default:
			return Session{}, fmt.Errorf("got error %q", err)
		}
		break
	}
	return Authenticate()
}

// ExchangeAuthCode exchanges an OAuth authorization code for tokens and returns a Session.
// This is useful when you already have the authorization code from a custom HTTP server.
func ExchangeAuthCode(code string) (Session, error) {
	resp, err := authenticateMSA(url.Values{
		"client_id":    {ClientID},
		"scope":        {scope},
		"redirect_uri": {RedirectURI.String()},
		"grant_type":   {"authorization_code"},
		"code":         {code},
	})
	if err != nil {
		return Session{}, fmt.Errorf("authenticate with MSA: %w", err)
	}
	Store.MSA.write(resp)

	return Authenticate()
}
