package discovery

import (
	"bytes"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// DiscoveredCamera represents a camera found via ONVIF
type DiscoveredCamera struct {
	Channel   int    `json:"channel"`
	Name      string `json:"name"`
	RTSPMain  string `json:"rtsp_main"`
	RTSPSub   string `json:"rtsp_sub,omitempty"`
	ProfileID string `json:"profile_id"`
}

// ONVIFClient communicates with an ONVIF device
type ONVIFClient struct {
	address  string
	username string
	password string
	client   *http.Client
}

func NewONVIFClient(address, username, password string) *ONVIFClient {
	return &ONVIFClient{
		address:  strings.TrimRight(address, "/"),
		username: username,
		password: password,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// wsSecurityHeader generates a WS-Security UsernameToken with PasswordDigest
func (c *ONVIFClient) wsSecurityHeader() string {
	nonce := make([]byte, 16)
	rand.Read(nonce)

	created := time.Now().UTC().Format(time.RFC3339Nano)

	// PasswordDigest = Base64(SHA1(nonce + created + password))
	h := sha1.New()
	h.Write(nonce)
	h.Write([]byte(created))
	h.Write([]byte(c.password))
	digest := base64.StdEncoding.EncodeToString(h.Sum(nil))
	nonceB64 := base64.StdEncoding.EncodeToString(nonce)

	return fmt.Sprintf(`<Security xmlns="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-secext-1.0.xsd">
      <UsernameToken>
        <Username>%s</Username>
        <Password Type="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-username-token-profile-1.0#PasswordDigest">%s</Password>
        <Nonce EncodingType="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-soap-message-security-1.0#Base64Binary">%s</Nonce>
        <Created xmlns="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-utility-1.0.xsd">%s</Created>
      </UsernameToken>
    </Security>`,
		c.username, digest, nonceB64, created)
}

// soapEnvelope wraps a SOAP body with WS-Security auth
func (c *ONVIFClient) soapEnvelope(body string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<Envelope xmlns="http://www.w3.org/2003/05/soap-envelope"
          xmlns:trt="http://www.onvif.org/ver10/media/wsdl"
          xmlns:tt="http://www.onvif.org/ver10/schema">
  <Header>
    %s
  </Header>
  <Body>
    %s
  </Body>
</Envelope>`, c.wsSecurityHeader(), body)
}

func (c *ONVIFClient) doSOAP(serviceURL, body string) ([]byte, error) {
	envelope := c.soapEnvelope(body)

	req, err := http.NewRequest(http.MethodPost, serviceURL, bytes.NewBufferString(envelope))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/soap+xml; charset=utf-8")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ONVIF request: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ONVIF error (status %d): %s", resp.StatusCode, string(data))
	}

	return data, nil
}

// Profile represents an ONVIF media profile
type Profile struct {
	Token string
	Name  string
}

// GetProfiles retrieves all media profiles from the device
func (c *ONVIFClient) GetProfiles() ([]Profile, error) {
	body := `<trt:GetProfiles/>`
	mediaURL := c.address + "/onvif/media_service"

	data, err := c.doSOAP(mediaURL, body)
	if err != nil {
		return nil, fmt.Errorf("GetProfiles: %w", err)
	}

	return parseProfiles(data)
}

// GetStreamURI gets the RTSP URI for a profile token
func (c *ONVIFClient) GetStreamURI(profileToken string) (string, error) {
	body := fmt.Sprintf(`<trt:GetStreamUri>
      <trt:StreamSetup>
        <tt:Stream>RTP-Unicast</tt:Stream>
        <tt:Transport><tt:Protocol>RTSP</tt:Protocol></tt:Transport>
      </trt:StreamSetup>
      <trt:ProfileToken>%s</trt:ProfileToken>
    </trt:GetStreamUri>`, profileToken)

	mediaURL := c.address + "/onvif/media_service"
	data, err := c.doSOAP(mediaURL, body)
	if err != nil {
		return "", fmt.Errorf("GetStreamUri: %w", err)
	}

	return parseStreamURI(data)
}

// DiscoverCameras probes the ONVIF device and returns all channels with stream URIs
func (c *ONVIFClient) DiscoverCameras(dvrHost string) ([]DiscoveredCamera, error) {
	profiles, err := c.GetProfiles()
	if err != nil {
		return nil, err
	}

	// Group profiles by channel: Profile_1_0 = ch1 main, Profile_1_1 = ch1 sub
	type channelStreams struct {
		name    string
		main    string
		sub     string
		mainTok string
	}
	channels := make(map[int]*channelStreams)

	for _, p := range profiles {
		ch, streamIdx := parseProfileToken(p.Token)
		if ch < 0 {
			continue
		}

		uri, err := c.GetStreamURI(p.Token)
		if err != nil {
			continue
		}

		// Rewrite internal IPs to configured DVR host
		if dvrHost != "" {
			uri = rewriteHost(uri, dvrHost)
		}

		cs, ok := channels[ch]
		if !ok {
			cs = &channelStreams{name: fmt.Sprintf("Camera %d", ch)}
			channels[ch] = cs
		}

		if streamIdx == 0 {
			cs.main = uri
			cs.mainTok = p.Token
			if p.Name != "" {
				cs.name = p.Name
			}
		} else {
			cs.sub = uri
		}
	}

	var result []DiscoveredCamera
	for ch, cs := range channels {
		if cs.main == "" {
			continue
		}
		result = append(result, DiscoveredCamera{
			Channel:   ch,
			Name:      cs.name,
			RTSPMain:  cs.main,
			RTSPSub:   cs.sub,
			ProfileID: cs.mainTok,
		})
	}

	return result, nil
}

// parseProfileToken extracts channel and stream index from e.g. "Profile_1_0"
func parseProfileToken(token string) (channel int, streamIdx int) {
	// Handle formats like "Profile_1_0", "Profile_12_1"
	parts := strings.Split(token, "_")
	if len(parts) < 3 {
		return -1, -1
	}

	var ch, idx int
	if _, err := fmt.Sscanf(parts[1], "%d", &ch); err != nil {
		return -1, -1
	}
	if _, err := fmt.Sscanf(parts[2], "%d", &idx); err != nil {
		return -1, -1
	}
	return ch, idx
}

// rewriteHost replaces the host in an RTSP URL
func rewriteHost(rtspURL, newHost string) string {
	u, err := url.Parse(rtspURL)
	if err != nil {
		return rtspURL
	}
	port := u.Port()
	if port != "" {
		u.Host = newHost + ":" + port
	} else {
		u.Host = newHost
	}
	return u.String()
}

// XML parsing helpers

type soapEnvelopeResp struct {
	XMLName xml.Name `xml:"Envelope"`
	Body    soapBody `xml:"Body"`
}

type soapBody struct {
	GetProfilesResponse struct {
		Profiles []xmlProfile `xml:"Profiles"`
	} `xml:"GetProfilesResponse"`
	GetStreamUriResponse struct {
		MediaUri struct {
			Uri string `xml:"Uri"`
		} `xml:"MediaUri"`
	} `xml:"GetStreamUriResponse"`
}

type xmlProfile struct {
	Token string `xml:"token,attr"`
	Name  string `xml:"Name"`
}

func parseProfiles(data []byte) ([]Profile, error) {
	var env soapEnvelopeResp
	if err := xml.Unmarshal(data, &env); err != nil {
		return nil, fmt.Errorf("parse profiles: %w", err)
	}

	var profiles []Profile
	for _, xp := range env.Body.GetProfilesResponse.Profiles {
		profiles = append(profiles, Profile{
			Token: xp.Token,
			Name:  xp.Name,
		})
	}
	return profiles, nil
}

func parseStreamURI(data []byte) (string, error) {
	var env soapEnvelopeResp
	if err := xml.Unmarshal(data, &env); err != nil {
		return "", fmt.Errorf("parse stream uri: %w", err)
	}

	uri := env.Body.GetStreamUriResponse.MediaUri.Uri
	if uri == "" {
		return "", fmt.Errorf("empty stream URI in response")
	}
	return uri, nil
}
