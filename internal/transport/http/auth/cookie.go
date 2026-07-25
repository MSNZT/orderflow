package auth

import (
	"net/http"
	"time"
)

type CookieManager struct {
	secureCookie bool
}

func NewCookieManager(isSecure bool) *CookieManager {
	return &CookieManager{secureCookie: isSecure}
}

const refreshCookieName = "refresh_token"
const refreshCookiePath = "/api/v1/auth"

func (cm *CookieManager) setRefreshToken(w http.ResponseWriter, refreshToken string, refreshTokenTTL time.Duration) {
	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookieName,
		Value:    refreshToken,
		Path:     refreshCookiePath,
		HttpOnly: true,
		Secure:   cm.secureCookie,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(refreshTokenTTL.Seconds()),
		Expires:  time.Now().Add(refreshTokenTTL),
	})
}

func (cm *CookieManager) clearRefreshCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookieName,
		Value:    "",
		Path:     refreshCookiePath,
		HttpOnly: true,
		Secure:   cm.secureCookie,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	})
}
