package api

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	httpapi "github.com/tiagorocha94/household-webapp/internal/api"
	"github.com/tiagorocha94/household-webapp/internal/auth"
	"github.com/tiagorocha94/household-webapp/internal/auth/service"
)

const (
	// sessionCookieName is the name of the cookie used to store the session token.
	sessionCookieName = "session_token"
	// maxLoginBodyBytes caps the login request body.
	maxLoginBodyBytes = 1 << 12
	// maxLoginFailures is how many failed logins one client address or one
	// username may have within loginFailureWindow before logins are refused.
	maxLoginFailures   = 10
	loginFailureWindow = 15 * time.Minute
	// LimitPruneInterval is how often PruneLimits is worth calling: a window
	// that has ended is of no use, and none lasts longer than this.
	LimitPruneInterval = loginFailureWindow
)

// Options configures the Handler.
type Options struct {
	// SecureCookies marks the session cookie Secure, so browsers only send it
	// over HTTPS. Leave it off for plain HTTP on localhost.
	SecureCookies bool
}

// Handler serves the authentication JSON API.
type Handler struct {
	log     *slog.Logger
	service *service.Service
	limiter *service.LoginLimiter
	// deviceLimiter counts failed enrolments and failed device tokens per
	// client address.
	deviceLimiter *service.LoginLimiter
	opts          Options
}

type apiUser struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
}

type apiLoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// PruneLimits forgets the failed attempts whose window has ended.
func (h *Handler) PruneLimits() {
	h.limiter.Prune()
	h.deviceLimiter.Prune()
}

// New creates a new Handler backed by the given Service, logging to log.
func New(svc *service.Service, log *slog.Logger, opts Options) *Handler {
	return &Handler{
		log:           log,
		service:       svc,
		limiter:       service.NewLoginLimiter(maxLoginFailures, loginFailureWindow),
		deviceLimiter: service.NewLoginLimiter(maxLoginFailures, loginFailureWindow),
		opts:          opts,
	}
}

// Login authenticates a JSON {"username","password"} body, sets the
// session cookie and returns the user. Bad credentials answer 401.
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxLoginBodyBytes)

	var req apiLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpapi.WriteError(w, http.StatusBadRequest, "Bad Request")
		return
	}

	username := strings.ToLower(strings.TrimSpace(req.Username))
	ipKey, userKey := "ip:"+clientIP(r), "user:"+username
	if ok, retryAfter := h.limiter.Allow(ipKey, userKey); !ok {
		w.Header().Set("Retry-After", strconv.Itoa(int(retryAfter.Seconds())+1))
		httpapi.WriteError(w, http.StatusTooManyRequests, "Too Many Requests")
		return
	}

	token, user, err := h.service.Authenticate(username, req.Password)
	if errors.Is(err, service.ErrInvalidCredentials) {
		h.limiter.Fail(ipKey, userKey)
		httpapi.WriteError(w, http.StatusUnauthorized, "Invalid credentials")
		return
	}
	if err != nil {
		h.log.Error("authenticate user", "username", username, "error", err)
		httpapi.WriteError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	// A success clears the username's failures but not the address's, so one
	// valid account cannot be used to reset the count for guessing others.
	h.limiter.Reset(userKey)
	h.setSessionCookie(w, token)
	_ = httpapi.WriteJSON(w, http.StatusOK, apiUser{ID: user.ID, Username: user.Username})
}

// Logout ends the session, if any, clears the cookie and answers 204.
// It is idempotent and does not require an active session. A request that
// the browser marks as coming from another site is refused with 403, so a
// foreign page cannot log the user out.
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	if isCrossSite(r) {
		httpapi.WriteError(w, http.StatusForbidden, "Forbidden")
		return
	}
	if cookie, err := r.Cookie(sessionCookieName); err == nil {
		if err := h.service.Logout(cookie.Value); err != nil {
			h.log.Error("end session", "error", err)
			httpapi.WriteError(w, http.StatusInternalServerError, "Internal Server Error")
			return
		}
	}
	h.clearSessionCookie(w)
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusNoContent)
}

// CurrentUser returns the authenticated user. It must run behind RequireAuth.
func (h *Handler) CurrentUser(r *http.Request) (any, error) {
	user := auth.UserFromContext(r.Context())
	return apiUser{ID: user.ID, Username: user.Username}, nil
}

func (h *Handler) setSessionCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   int(service.SessionTTL / time.Second),
		HttpOnly: true,
		Secure:   h.opts.SecureCookies,
		SameSite: http.SameSiteLaxMode,
	})
}

func (h *Handler) clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   h.opts.SecureCookies,
		SameSite: http.SameSiteLaxMode,
	})
}

// clientIP returns the address the request came from. It does not trust
// forwarding headers: the server is meant to be reached directly.
func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// isCrossSite reports whether a browser says the request was started by a
// different site. It uses Sec-Fetch-Site when the browser sends it, and the
// Origin header otherwise. Requests with neither, such as curl, are allowed.
func isCrossSite(r *http.Request) bool {
	if site := r.Header.Get("Sec-Fetch-Site"); site != "" {
		return site != "same-origin" && site != "none"
	}
	if origin := r.Header.Get("Origin"); origin != "" {
		u, err := url.Parse(origin)
		return err != nil || u.Host != r.Host
	}
	return false
}
