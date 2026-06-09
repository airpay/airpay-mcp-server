package qrlink

import (
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"
)

// Handler serves /qr/{token}/preview and /qr/{token}/download endpoints.
type Handler struct {
	masterKey []byte
	logger    *log.Logger
}

func NewHandler(masterKey []byte, logger *log.Logger) *Handler {
	return &Handler{masterKey: masterKey, logger: logger}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/qr/")
	idx := strings.LastIndex(path, "/")
	if idx < 1 {
		http.Error(w, "invalid QR link", http.StatusBadRequest)
		return
	}
	token, action := path[:idx], path[idx+1:]
	payload, err := DecryptToken(token, h.masterKey)
	if err != nil {
		h.logger.Printf("[qrlink] token error: %v (path=%s)", err, r.URL.Path)
		if errors.Is(err, ErrExpired) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusGone)
			fmt.Fprint(w, expiredHTML)
			return
		}
		http.Error(w, "invalid or tampered QR link", http.StatusBadRequest)
		return
	}
	switch action {
	case "preview":
		h.handlePreview(w, r, token, payload)
	default:
		http.NotFound(w, r)
	}
}

func (h *Handler) handlePreview(w http.ResponseWriter, r *http.Request, token string, payload *QRPayload) {
	svgStr, err := GenerateQRCodeSVG(payload.QRCodeString, 400)
	if err != nil {
		h.logger.Printf("[qrlink] QR SVG generation failed: %v", err)
		http.Error(w, "failed to generate QR code", http.StatusInternalServerError)
		return
	}
	h.logger.Printf("[qrlink] QR SVG length: %d bytes", len(svgStr))
	data := previewData{
		QRSVG:            template.HTML(svgStr),
		UPIDeepLink:      template.URL(payload.QRCodeString),
		APTransactionID:  payload.APTransactionID,
		Amount:           payload.Amount,
		OrderID:          payload.OrderID,
		MerchantID:       payload.MerchantID,
		ExpiryTimestamp:  payload.ExpiresAt,
		DownloadPagePath: template.URL("/qr/" + token + "/download"),
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'unsafe-inline'; img-src 'self' data: blob:; script-src 'self' 'unsafe-inline' 'unsafe-eval'")
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Referrer-Policy", "no-referrer")
	if err := previewTmpl.Execute(w, data); err != nil {
		h.logger.Printf("[qrlink] template render failed: %v", err)
	}
	h.logger.Printf("[qrlink] preview served: txn=%s", payload.APTransactionID)
}

type previewData struct {
	QRSVG            template.HTML
	UPIDeepLink      template.URL
	APTransactionID  string
	Amount           string
	OrderID          string
	MerchantID       string
	ExpiryTimestamp  int64
	DownloadPagePath template.URL
}

const expiredHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>QR Expired</title>
<style>
  body{font-family:sans-serif;display:flex;align-items:center;justify-content:center;
       min-height:100vh;margin:0;background:#f5f7fa;}
  .card{background:white;padding:40px 32px;border-radius:16px;text-align:center;
        box-shadow:0 4px 24px rgba(0,0,0,0.08);max-width:360px;width:100%;}
  h2{color:#ef4444;margin-bottom:12px;font-size:20px;}
  p{color:#6b7280;font-size:14px;line-height:1.6;}
</style>
</head>
<body>
<div class="card">
  <h2>QR Code Expired</h2>
  <p>This payment QR link has expired.<br>Please generate a new QR code.</p>
</div>
</body>
</html>`
