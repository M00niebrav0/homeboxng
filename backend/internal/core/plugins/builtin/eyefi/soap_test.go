package eyefi

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/xml"
	"strings"
	"testing"

	"github.com/rs/zerolog"
)

func newTestHandler() *SOAPHandler {
	return NewSOAPHandler("abc123def456", "00:1A:7D:DA:71:FF", zerolog.Nop())
}

func TestNewSOAPHandler_NormalizesMAC(t *testing.T) {
	h := NewSOAPHandler("key", "00:1A:7D:DA:71:FF", zerolog.Nop())
	if h.cardMAC != "001a7dda71ff" {
		t.Errorf("MAC = %q, want %q", h.cardMAC, "001a7dda71ff")
	}
}

func TestParseRequest_StartSession(t *testing.T) {
	h := newTestHandler()

	xmlData := `<?xml version="1.0"?>
<SOAP-ENV:Envelope xmlns:SOAP-ENV="http://schemas.xmlsoap.org/soap/envelope/">
  <SOAP-ENV:Body>
    <StartSession>
      <macaddress>001a7dda71ff</macaddress>
      <cnonce>abcdef1234567890</cnonce>
      <transfermode>2</transfermode>
      <transfermodetimestamp>1234567890</transfermodetimestamp>
    </StartSession>
  </SOAP-ENV:Body>
</SOAP-ENV:Envelope>`

	body, err := h.ParseRequest([]byte(xmlData))
	if err != nil {
		t.Fatalf("ParseRequest() error = %v", err)
	}
	if body.StartSession == nil {
		t.Fatal("expected StartSession to be parsed")
	}
	if body.StartSession.MacAddress != "001a7dda71ff" {
		t.Errorf("MAC = %q, want %q", body.StartSession.MacAddress, "001a7dda71ff")
	}
	if body.StartSession.CNonce != "abcdef1234567890" {
		t.Errorf("CNonce = %q", body.StartSession.CNonce)
	}
}

func TestParseRequest_GetPhotoStatus(t *testing.T) {
	h := newTestHandler()

	xmlData := `<?xml version="1.0"?>
<SOAP-ENV:Envelope xmlns:SOAP-ENV="http://schemas.xmlsoap.org/soap/envelope/">
  <SOAP-ENV:Body>
    <GetPhotoStatus>
      <macaddress>001a7dda71ff</macaddress>
      <credential>abc123</credential>
      <filename>DSC_0042.JPG</filename>
      <filesize>2500000</filesize>
      <filesignature>abc123</filesignature>
    </GetPhotoStatus>
  </SOAP-ENV:Body>
</SOAP-ENV:Envelope>`

	body, err := h.ParseRequest([]byte(xmlData))
	if err != nil {
		t.Fatalf("ParseRequest() error = %v", err)
	}
	if body.GetPhotoStatus == nil {
		t.Fatal("expected GetPhotoStatus to be parsed")
	}
	if body.GetPhotoStatus.Filename != "DSC_0042.JPG" {
		t.Errorf("Filename = %q", body.GetPhotoStatus.Filename)
	}
	if body.GetPhotoStatus.FileSize != 2500000 {
		t.Errorf("FileSize = %d", body.GetPhotoStatus.FileSize)
	}
}

func TestParseRequest_UploadPhoto(t *testing.T) {
	h := newTestHandler()

	xmlData := `<?xml version="1.0"?>
<SOAP-ENV:Envelope xmlns:SOAP-ENV="http://schemas.xmlsoap.org/soap/envelope/">
  <SOAP-ENV:Body>
    <UploadPhoto>
      <macaddress>001a7dda71ff</macaddress>
      <filename>IMG_0001.JPG</filename>
      <filesize>1500000</filesize>
      <filesignature>sig123</filesignature>
    </UploadPhoto>
  </SOAP-ENV:Body>
</SOAP-ENV:Envelope>`

	body, err := h.ParseRequest([]byte(xmlData))
	if err != nil {
		t.Fatalf("ParseRequest() error = %v", err)
	}
	if body.UploadPhoto == nil {
		t.Fatal("expected UploadPhoto to be parsed")
	}
	if body.UploadPhoto.Filename != "IMG_0001.JPG" {
		t.Errorf("Filename = %q", body.UploadPhoto.Filename)
	}
}

func TestParseRequest_InvalidXML(t *testing.T) {
	h := newTestHandler()
	_, err := h.ParseRequest([]byte("not xml"))
	if err == nil {
		t.Error("expected error for invalid XML")
	}
}

func TestParseRequest_EmptyBody(t *testing.T) {
	h := newTestHandler()
	xmlData := `<SOAP-ENV:Envelope xmlns:SOAP-ENV="http://schemas.xmlsoap.org/soap/envelope/"></SOAP-ENV:Envelope>`

	_, err := h.ParseRequest([]byte(xmlData))
	if err == nil {
		t.Error("expected error for empty SOAP body")
	}
}

func TestHandleStartSession(t *testing.T) {
	h := newTestHandler()

	req := &StartSessionRequest{
		MacAddress:   "001a7dda71ff",
		CNonce:       "testcnonce123456",
		TransferMode: 2,
	}

	resp, err := h.HandleStartSession(req)
	if err != nil {
		t.Fatalf("HandleStartSession() error = %v", err)
	}

	if resp.Credential == "" {
		t.Error("expected non-empty credential")
	}
	if resp.SNonce == "" {
		t.Error("expected non-empty snonce")
	}
	if resp.TransferMode != 2 {
		t.Errorf("TransferMode = %d, want 2", resp.TransferMode)
	}

	// Verify credential is correct MD5
	expected := computeCredential("001a7dda71ff", "testcnonce123456", "abc123def456")
	if resp.Credential != expected {
		t.Errorf("credential = %q, want %q", resp.Credential, expected)
	}
}

func TestHandleStartSession_WrongMAC(t *testing.T) {
	h := newTestHandler()

	req := &StartSessionRequest{
		MacAddress: "ff:ff:ff:ff:ff:ff",
		CNonce:     "test",
	}

	_, err := h.HandleStartSession(req)
	if err == nil {
		t.Error("expected error for wrong MAC address")
	}
}

func TestHandleStartSession_EmptyConfiguredMAC(t *testing.T) {
	// If no MAC is configured, accept any card
	h := NewSOAPHandler("key", "", zerolog.Nop())

	req := &StartSessionRequest{
		MacAddress: "001a7dda71ff",
		CNonce:     "test",
	}

	resp, err := h.HandleStartSession(req)
	if err != nil {
		t.Fatalf("expected no error when MAC not configured: %v", err)
	}
	if resp == nil {
		t.Fatal("expected response")
	}
}

func TestHandleGetPhotoStatus(t *testing.T) {
	h := newTestHandler()

	req := &GetPhotoStatusRequest{
		Filename: "DSC_0042.JPG",
		FileSize: 2500000,
	}

	resp := h.HandleGetPhotoStatus(req)
	if resp.FileID != 1 {
		t.Errorf("FileID = %d, want 1", resp.FileID)
	}
	if resp.Offset != 0 {
		t.Errorf("Offset = %d, want 0", resp.Offset)
	}
}

func TestComputeCredential(t *testing.T) {
	mac := "001a7dda71ff"
	cnonce := "testcnonce"
	key := "abc123"

	result := computeCredential(mac, cnonce, key)

	// Verify it's a valid 32-char hex string
	if len(result) != 32 {
		t.Fatalf("credential length = %d, want 32", len(result))
	}
	_, err := hex.DecodeString(result)
	if err != nil {
		t.Fatalf("credential is not valid hex: %v", err)
	}

	// Verify deterministic
	result2 := computeCredential(mac, cnonce, key)
	if result != result2 {
		t.Error("expected same credential for same inputs")
	}

	// Verify correct MD5
	data := mac + cnonce + key
	hash := md5.Sum([]byte(data))
	expected := hex.EncodeToString(hash[:])
	if result != expected {
		t.Errorf("credential = %q, want %q", result, expected)
	}
}

func TestGenerateNonce(t *testing.T) {
	nonce := generateNonce()
	if len(nonce) != 32 {
		t.Errorf("nonce length = %d, want 32", len(nonce))
	}
	_, err := hex.DecodeString(nonce)
	if err != nil {
		t.Fatalf("nonce is not valid hex: %v", err)
	}
}

func TestWrapResponse(t *testing.T) {
	resp := &StartSessionResponse{
		Credential:   "abc123",
		SNonce:       "def456",
		TransferMode: 2,
	}

	data, err := WrapResponse(resp)
	if err != nil {
		t.Fatalf("WrapResponse() error = %v", err)
	}

	s := string(data)

	// Should contain SOAP envelope
	if !strings.Contains(s, "SOAP-ENV:Envelope") {
		t.Error("expected SOAP envelope in output")
	}
	if !strings.Contains(s, "SOAP-ENV:Body") {
		t.Error("expected SOAP body in output")
	}
	if !strings.Contains(s, "<credential>abc123</credential>") {
		t.Error("expected credential in output")
	}
	if !strings.Contains(s, `<?xml version="1.0"`) {
		t.Error("expected XML declaration")
	}
}

func TestWrapResponse_Roundtrip(t *testing.T) {
	resp := &GetPhotoStatusResponse{
		FileID: 42,
		Offset: 0,
	}

	data, err := WrapResponse(resp)
	if err != nil {
		t.Fatalf("WrapResponse() error = %v", err)
	}

	// Parse the wrapped response back
	var envelope struct {
		XMLName xml.Name `xml:"Envelope"`
		Body    struct {
			Response GetPhotoStatusResponse `xml:"GetPhotoStatusResponse"`
		} `xml:"Body"`
	}

	if err := xml.Unmarshal(data, &envelope); err != nil {
		t.Fatalf("roundtrip parse error: %v", err)
	}

	if envelope.Body.Response.FileID != 42 {
		t.Errorf("FileID = %d, want 42", envelope.Body.Response.FileID)
	}
}
