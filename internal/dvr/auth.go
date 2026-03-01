package dvr

import (
	"crypto/md5"
	"crypto/sha512"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"sync"
	"time"
)

type Session struct {
	ID        string
	Token     string
	Host      string
	Cookies   []*http.Cookie
	ExpiresAt time.Time
}

type Authenticator struct {
	mu       sync.Mutex
	sessions map[string]*Session // keyed by "host:username"
}

func NewAuthenticator() *Authenticator {
	return &Authenticator{
		sessions: make(map[string]*Session),
	}
}

func (a *Authenticator) GetSession(host, username, password string) (*Session, error) {
	key := host + ":" + username

	a.mu.Lock()
	if s, ok := a.sessions[key]; ok && time.Now().Before(s.ExpiresAt) {
		a.mu.Unlock()
		return s, nil
	}
	a.mu.Unlock()

	session, err := a.login(host, username, password)
	if err != nil {
		return nil, err
	}

	a.mu.Lock()
	a.sessions[key] = session
	a.mu.Unlock()

	return session, nil
}

// XML structures for DVR API

type xmlResponse struct {
	Status    string `xml:"status"`
	ErrorCode string `xml:"errorCode"`
	Content   xmlContent `xml:"content"`
}

type xmlContent struct {
	Nonce     string `xml:"nonce"`
	SessionID string `xml:"sessionId"`
	Token     string `xml:"token"`
	UserID    string `xml:"userId"`
	SessionKey string `xml:"sessionKey"`
}

const xmlHeader = `<?xml version="1.0" encoding="utf-8" ?>`
const protocolVer = "1.0"
const systemType = "NVMS-9000"

func emptyRequest(token string) string {
	return fmt.Sprintf(
		`%s<request version="%s" systemType="%s" clientType="WEB"><token>%s</token></request>`,
		xmlHeader, protocolVer, systemType, token,
	)
}

func loginRequest(token, username, hashedPassword string) string {
	return fmt.Sprintf(
		`%s<request version="%s" systemType="%s" clientType="WEB"><token>%s</token>`+
			`<content><userName><![CDATA[%s]]></userName>`+
			`<password><![CDATA[%s]]></password></content></request>`,
		xmlHeader, protocolVer, systemType, token, username, hashedPassword,
	)
}

func (a *Authenticator) login(host, username, password string) (*Session, error) {
	jar, _ := cookiejar.New(nil)
	client := &http.Client{
		Timeout: 10 * time.Second,
		Jar:     jar,
	}

	origin := fmt.Sprintf("http://%s", host)

	// Step 1: reqLogin — get nonce, sessionId, token
	reqLoginURL := origin + "/reqLogin"
	reqBody := emptyRequest("")

	req, err := http.NewRequest("POST", reqLoginURL, strings.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("reqLogin new request failed: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("Origin", origin)
	req.Header.Set("Referer", origin+"/")
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("reqLogin request failed: %w", err)
	}
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("reqLogin read body failed: %w", err)
	}

	var reqLoginResp xmlResponse
	if err := xml.Unmarshal(body, &reqLoginResp); err != nil {
		return nil, fmt.Errorf("reqLogin parse XML failed: %w (body: %s)", err, string(body))
	}
	if reqLoginResp.Status != "success" {
		return nil, fmt.Errorf("reqLogin failed: status=%s errorCode=%s", reqLoginResp.Status, reqLoginResp.ErrorCode)
	}

	nonce := reqLoginResp.Content.Nonce
	sessionID := reqLoginResp.Content.SessionID
	token := reqLoginResp.Content.Token

	// Strip braces from sessionId if present
	sessionID = strings.TrimPrefix(sessionID, "{")
	sessionID = strings.TrimSuffix(sessionID, "}")

	// Step 2: Hash password — SHA512(MD5(password) + "#" + nonce)
	hashedPassword := hashPassword(password, nonce)

	// Step 3: doLogin — authenticate with hashed password
	doLoginURL := origin + "/doLogin"
	loginBody := loginRequest(token, username, hashedPassword)

	req, err = http.NewRequest("POST", doLoginURL, strings.NewReader(loginBody))
	if err != nil {
		return nil, fmt.Errorf("doLogin new request failed: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("Origin", origin)
	req.Header.Set("Referer", origin+"/")
	resp, err = client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("doLogin request failed: %w", err)
	}
	body, err = io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("doLogin read body failed: %w", err)
	}

	var doLoginResp xmlResponse
	if err := xml.Unmarshal(body, &doLoginResp); err != nil {
		return nil, fmt.Errorf("doLogin parse XML failed: %w (body: %s)", err, string(body))
	}
	if doLoginResp.Status != "success" {
		return nil, fmt.Errorf("doLogin failed: status=%s errorCode=%s", doLoginResp.Status, doLoginResp.ErrorCode)
	}

	return &Session{
		ID:        sessionID,
		Token:     token,
		Host:      host,
		Cookies:   resp.Cookies(),
		ExpiresAt: time.Now().Add(25 * time.Minute),
	}, nil
}

func hashPassword(password, nonce string) string {
	// MD5(password) — UPPERCASE hex to match SparkMD5.hash() on DVR
	md5Hash := md5.Sum([]byte(password))
	md5Hex := strings.ToUpper(hex.EncodeToString(md5Hash[:]))

	// SHA512(md5hex + "#" + nonce)
	sha := sha512.Sum512([]byte(md5Hex + "#" + nonce))
	return hex.EncodeToString(sha[:])
}

func (a *Authenticator) Invalidate(host, username string) {
	key := host + ":" + username
	a.mu.Lock()
	delete(a.sessions, key)
	a.mu.Unlock()
}
