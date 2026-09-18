package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const Cookie = "acahti"

type Service struct {
	Secret []byte
	TTL    time.Duration
	Admin  string
}

func New(secret []byte, admin string) *Service {
	return &Service{Secret: secret, TTL: 30 * 24 * time.Hour, Admin: admin}
}

func (s *Service) Issue(user string) string {
	return s.IssueFor(user, s.TTL)
}

func (s *Service) IssueFor(user string, ttl time.Duration) string {
	exp := strconv.FormatInt(time.Now().Add(ttl).Unix(), 10)
	mac := hmac.New(sha256.New, s.Secret)
	mac.Write([]byte(user + "|" + exp))
	return base64.RawURLEncoding.EncodeToString([]byte(user + "|" + exp + "|" + hex.EncodeToString(mac.Sum(nil))))
}

func (s *Service) Parse(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", false
	}
	b, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return "", false
	}
	parts := strings.Split(string(b), "|")
	if len(parts) != 3 {
		return "", false
	}
	user, exp, sig := parts[0], parts[1], parts[2]
	mac := hmac.New(sha256.New, s.Secret)
	mac.Write([]byte(user + "|" + exp))
	if !hmac.Equal([]byte(sig), []byte(hex.EncodeToString(mac.Sum(nil)))) {
		return "", false
	}
	until, err := strconv.ParseInt(exp, 10, 64)
	if err != nil || time.Now().Unix() > until || user == "" {
		return "", false
	}
	return user, true
}

func (s *Service) CookieUser(r *http.Request) string {
	c, err := r.Cookie(Cookie)
	if err != nil {
		return ""
	}
	user, ok := s.Parse(c.Value)
	if !ok {
		return ""
	}
	return user
}

func (s *Service) SetCookie(w http.ResponseWriter, user string) {
	http.SetCookie(w, &http.Cookie{
		Name:     Cookie,
		Value:    s.Issue(user),
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(s.TTL.Seconds()),
	})
}

func (s *Service) ClearCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{Name: Cookie, Path: "/", MaxAge: -1})
}

func (s *Service) IsAdmin(user string) bool {
	return user != "" && user == s.Admin
}

func Bearer(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if strings.HasPrefix(strings.ToLower(h), "bearer ") {
		return strings.TrimSpace(h[7:])
	}
	if strings.HasPrefix(strings.ToLower(h), "token ") {
		return strings.TrimSpace(h[6:])
	}
	return ""
}
