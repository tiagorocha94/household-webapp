package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	httpapi "github.com/tiagorocha94/household-webapp/internal/api"
	"github.com/tiagorocha94/household-webapp/internal/auth"
)

// maxEnrolBodyBytes caps the enrolment request body.
const maxEnrolBodyBytes = 1 << 10

type apiEnrolRequest struct {
	Code string `json:"code"`
}

type apiEnrolResponse struct {
	Token      string `json:"token"`
	Username   string `json:"username"`
	DeviceName string `json:"deviceName"`
}

// Enrol exchanges a one-time enrolment code for a long-lived device token. The
// answer holds the token, shown once, and the username of the person the
// device writes for. A wrong, expired or used code answers 401, and repeated
// failures from one address answer 429, as for logins.
func (h *Handler) Enrol(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxEnrolBodyBytes)

	var req apiEnrolRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.Code) == "" {
		httpapi.WriteError(w, http.StatusBadRequest, "Bad Request")
		return
	}

	key := "ip:" + clientIP(r)
	if !h.allowDeviceAttempt(w, key) {
		return
	}

	token, device, err := h.service.Enrol(req.Code)
	if errors.Is(err, auth.ErrInvalidEnrolCode) {
		h.deviceLimiter.Fail(key)
		httpapi.WriteError(w, http.StatusUnauthorized, "Invalid or expired code")
		return
	}
	if err != nil {
		h.log.Error("enrol device", "error", err)
		httpapi.WriteError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	h.log.Info("device enrolled", "device", device.Name, "username", device.Username)
	_ = httpapi.WriteJSON(w, http.StatusOK, apiEnrolResponse{Token: token, Username: device.Username, DeviceName: device.Name})
}

// RequireDevice answers 401 unless the request carries the token of an
// enrolled, not revoked device in an "Authorization: Bearer" header, and
// otherwise stores the device and its person in the request context. It does
// not look at the session cookie, so a browser login cannot write as a device.
// Failed attempts are counted per client address, and an address with too
// many answers 429 until the window ends.
func (h *Handler) RequireDevice(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := "ip:" + clientIP(r)
		if !h.allowDeviceAttempt(w, key) {
			return
		}

		token, ok := bearerToken(r)
		var device *auth.Device
		var user *auth.User
		if ok {
			var err error
			device, user, err = h.service.DeviceByToken(token)
			if err != nil {
				h.log.Error("check device token", "error", err)
				httpapi.WriteError(w, http.StatusInternalServerError, "Internal Server Error")
				return
			}
		}
		if device == nil {
			h.deviceLimiter.Fail(key)
			w.Header().Set("WWW-Authenticate", "Bearer")
			httpapi.WriteError(w, http.StatusUnauthorized, "Unauthorized")
			return
		}

		ctx := auth.WithDevice(auth.WithUser(r.Context(), user), device)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireSessionOrDevice lets a read-only route be reached by the web interface
// (the session cookie) and by an enrolled phone (the bearer token). A request
// that carries a bearer token is checked as a device, whatever cookies come
// with it, so a wrong token is refused and counts toward the device limits.
// Without a token it is checked as a session, exactly as RequireAuth does.
func (h *Handler) RequireSessionOrDevice(next http.Handler) http.Handler {
	asDevice := h.RequireDevice(next)
	asSession := RequireAuth(h.service, h.log, next)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := bearerToken(r); ok {
			asDevice.ServeHTTP(w, r)
			return
		}
		asSession.ServeHTTP(w, r)
	})
}

// allowDeviceAttempt answers 429 with Retry-After and returns false when key
// has had too many failed device attempts.
func (h *Handler) allowDeviceAttempt(w http.ResponseWriter, key string) bool {
	ok, retryAfter := h.deviceLimiter.Allow(key)
	if !ok {
		w.Header().Set("Retry-After", strconv.Itoa(int(retryAfter.Seconds())+1))
		httpapi.WriteError(w, http.StatusTooManyRequests, "Too Many Requests")
	}
	return ok
}

// bearerToken returns the token of an "Authorization: Bearer <token>" header.
func bearerToken(r *http.Request) (string, bool) {
	scheme, token, found := strings.Cut(r.Header.Get("Authorization"), " ")
	token = strings.TrimSpace(token)
	if !found || !strings.EqualFold(scheme, "Bearer") || token == "" {
		return "", false
	}
	return token, true
}
