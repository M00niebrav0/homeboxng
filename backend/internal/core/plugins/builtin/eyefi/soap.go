package eyefi

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"strings"

	"github.com/rs/zerolog"
)

// SOAPEnvelope wraps SOAP request/response bodies.
type SOAPEnvelope struct {
	XMLName xml.Name  `xml:"Envelope"`
	Body    *SOAPBody `xml:"Body"`
}

// SOAPBody contains the SOAP operation.
type SOAPBody struct {
	StartSession   *StartSessionRequest   `xml:"StartSession,omitempty"`
	GetPhotoStatus *GetPhotoStatusRequest `xml:"GetPhotoStatus,omitempty"`
	UploadPhoto    *UploadPhotoRequest    `xml:"UploadPhoto,omitempty"`
}

// StartSessionRequest is sent by the Eye-Fi card to initiate a session.
type StartSessionRequest struct {
	MacAddress   string `xml:"macaddress"`
	CNonce       string `xml:"cnonce"`
	TransferMode int    `xml:"transfermode"`
	TransferHTTPS int   `xml:"transfermodetimestamp"`
}

// StartSessionResponse authenticates the server to the card.
type StartSessionResponse struct {
	XMLName        xml.Name `xml:"StartSessionResponse"`
	Credential     string   `xml:"credential"`
	SNonce         string   `xml:"snonce"`
	TransferMode   int      `xml:"transfermode"`
	TransferHTTPS  int      `xml:"transfermodetimestamp"`
	UpathReply     int      `xml:"upsyncallowed"`
}

// GetPhotoStatusRequest checks if the server wants the photo.
type GetPhotoStatusRequest struct {
	MacAddress string `xml:"macaddress"`
	Credential string `xml:"credential"`
	Filename   string `xml:"filename"`
	FileSize   int64  `xml:"filesize"`
	FileSig    string `xml:"filesignature"`
}

// GetPhotoStatusResponse tells the card whether to upload.
type GetPhotoStatusResponse struct {
	XMLName  xml.Name `xml:"GetPhotoStatusResponse"`
	FileID   int      `xml:"fileid"`
	Offset   int64    `xml:"offset"`
}

// UploadPhotoRequest is the metadata accompanying an uploaded photo.
type UploadPhotoRequest struct {
	MacAddress string `xml:"macaddress"`
	Filename   string `xml:"filename"`
	FileSize   int64  `xml:"filesize"`
	FileSig    string `xml:"filesignature"`
}

// UploadPhotoResponse acknowledges the upload.
type UploadPhotoResponse struct {
	XMLName xml.Name `xml:"UploadPhotoResponse"`
	Success bool     `xml:"success"`
}

// SOAPHandler processes Eye-Fi SOAP protocol messages.
type SOAPHandler struct {
	uploadKey string
	cardMAC   string
	logger    zerolog.Logger
}

// NewSOAPHandler creates a SOAP handler with the card's credentials.
func NewSOAPHandler(uploadKey, cardMAC string, logger zerolog.Logger) *SOAPHandler {
	return &SOAPHandler{
		uploadKey: uploadKey,
		cardMAC:   strings.ToLower(strings.ReplaceAll(cardMAC, ":", "")),
		logger:    logger.With().Str("component", "eyefi-soap").Logger(),
	}
}

// ParseRequest parses a SOAP XML request body.
func (h *SOAPHandler) ParseRequest(data []byte) (*SOAPBody, error) {
	var envelope SOAPEnvelope
	if err := xml.Unmarshal(data, &envelope); err != nil {
		return nil, fmt.Errorf("parsing SOAP envelope: %w", err)
	}
	if envelope.Body == nil {
		return nil, fmt.Errorf("empty SOAP body")
	}
	return envelope.Body, nil
}

// HandleStartSession processes the authentication handshake.
// Eye-Fi uses MD5(MAC + uploadKey + cnonce) for mutual authentication.
func (h *SOAPHandler) HandleStartSession(req *StartSessionRequest) (*StartSessionResponse, error) {
	mac := strings.ToLower(strings.ReplaceAll(req.MacAddress, ":", ""))

	if h.cardMAC != "" && mac != h.cardMAC {
		return nil, fmt.Errorf("unexpected card MAC: %s (expected %s)", mac, h.cardMAC)
	}

	// Generate server nonce
	snonce := generateNonce()

	// Calculate credential: MD5(MAC + cnonce + uploadKey)
	credential := computeCredential(mac, req.CNonce, h.uploadKey)

	h.logger.Info().
		Str("mac", mac).
		Str("cnonce", req.CNonce).
		Msg("eye-fi card started session")

	return &StartSessionResponse{
		Credential:    credential,
		SNonce:        snonce,
		TransferMode:  req.TransferMode,
		TransferHTTPS: req.TransferHTTPS,
		UpathReply:    0,
	}, nil
}

// HandleGetPhotoStatus checks if the server wants the photo (always yes).
func (h *SOAPHandler) HandleGetPhotoStatus(req *GetPhotoStatusRequest) *GetPhotoStatusResponse {
	h.logger.Debug().
		Str("filename", req.Filename).
		Int64("size", req.FileSize).
		Msg("photo status check")

	return &GetPhotoStatusResponse{
		FileID: 1,
		Offset: 0,
	}
}

// WrapResponse wraps a SOAP response in an envelope.
func WrapResponse(body any) ([]byte, error) {
	inner, err := xml.MarshalIndent(body, "    ", "  ")
	if err != nil {
		return nil, err
	}

	// Build SOAP envelope manually for Eye-Fi compatibility
	var sb strings.Builder
	sb.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
	sb.WriteString("\n")
	sb.WriteString(`<SOAP-ENV:Envelope xmlns:SOAP-ENV="http://schemas.xmlsoap.org/soap/envelope/">`)
	sb.WriteString("\n  <SOAP-ENV:Body>\n")
	sb.Write(inner)
	sb.WriteString("\n  </SOAP-ENV:Body>\n")
	sb.WriteString("</SOAP-ENV:Envelope>")

	return []byte(sb.String()), nil
}

// computeCredential calculates the Eye-Fi MD5 credential.
func computeCredential(mac, cnonce, uploadKey string) string {
	data := mac + cnonce + uploadKey
	hash := md5.Sum([]byte(data))
	return hex.EncodeToString(hash[:])
}

// generateNonce creates a random hex nonce for the SOAP session.
func generateNonce() string {
	// Simple nonce - in production, use crypto/rand
	data := fmt.Sprintf("%d", md5.Sum([]byte("homeboxng-eyefi-snonce")))
	hash := md5.Sum([]byte(data))
	return hex.EncodeToString(hash[:])
}
