package service

import (
	"crypto/rand"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/tiagorocha94/household-webapp/internal/apperr"
	"github.com/tiagorocha94/household-webapp/internal/auth"
)

const (
	// EnrolCodeTTL is how long an enrolment code can be exchanged for a token.
	EnrolCodeTTL = 15 * time.Minute
	// deviceTouchInterval is the least time between two last-seen updates of
	// one device.
	deviceTouchInterval = time.Minute
	// enrolCodeLength is the number of characters in an enrolment code, drawn
	// from enrolCodeAlphabet (32 symbols, so 50 bits).
	enrolCodeLength = 10
	// enrolCodeAlphabet leaves out the characters that are easy to mix up when
	// typing a code: 0 and O, 1 and I.
	enrolCodeAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
)

// deviceNamePattern allows short names a person can type and read in a list.
var deviceNamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9 ._-]{0,39}$`)

// AddDevice makes a pending device named name for the user and returns the
// one-time enrolment code, shown to the person once, and when it expires. The
// name is trimmed. It returns an apperr.ErrBadRequest error for a name that is
// not 1 to 40 letters, digits, spaces, dots, dashes or underscores, and the
// auth errors ErrUnknownUser, ErrUserHasDevice and ErrDeviceNameTaken.
func (s *Service) AddDevice(username, name string) (string, time.Time, error) {
	name = strings.TrimSpace(name)
	if !deviceNamePattern.MatchString(name) {
		return "", time.Time{}, fmt.Errorf("device name %q must be 1 to 40 letters, digits, spaces, dots, dashes or underscores: %w", name, apperr.ErrBadRequest)
	}
	code, err := newEnrolCode()
	if err != nil {
		return "", time.Time{}, err
	}
	now := s.now()
	expires := now.Add(EnrolCodeTTL)
	if _, err := s.store.CreateDevice(strings.TrimSpace(username), name, hashToken(normalizeEnrolCode(code)), now, expires); err != nil {
		return "", time.Time{}, err
	}
	return code, expires, nil
}

// Enrol exchanges an enrolment code for a long-lived device token, which is
// returned once and stored only as a hash. The code is used up. A wrong,
// expired or used code returns auth.ErrInvalidEnrolCode.
func (s *Service) Enrol(code string) (string, *auth.Device, error) {
	token, err := generateToken()
	if err != nil {
		return "", nil, fmt.Errorf("generate device token: %w", err)
	}
	device, _, err := s.store.EnrolDevice(hashToken(normalizeEnrolCode(code)), hashToken(token), s.now())
	if err != nil {
		return "", nil, err
	}
	return token, device, nil
}

// DeviceByToken returns the enrolled, not revoked device that owns the token
// and its user, or nil, nil for any other token. It records the use.
func (s *Service) DeviceByToken(token string) (*auth.Device, *auth.User, error) {
	device, user, err := s.store.GetDeviceByToken(hashToken(token))
	if err != nil || device == nil {
		return nil, nil, err
	}
	now := s.now()
	if err := s.store.TouchDevice(device.ID, now, now.Add(-deviceTouchInterval)); err != nil {
		return nil, nil, err
	}
	return device, user, nil
}

// ListDevices returns every device, revoked ones included.
func (s *Service) ListDevices() ([]auth.Device, error) {
	return s.store.ListDevices()
}

// RevokeDevice stops the device named name from working. It returns
// auth.ErrUnknownDevice when no such device is active.
func (s *Service) RevokeDevice(name string) error {
	return s.store.RevokeDevice(strings.TrimSpace(name), s.now())
}

// newEnrolCode returns a random code such as K7QD2-M9XWP.
func newEnrolCode() (string, error) {
	b := make([]byte, enrolCodeLength)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate enrolment code: %w", err)
	}
	var sb strings.Builder
	for i, v := range b {
		if i == enrolCodeLength/2 {
			sb.WriteByte('-')
		}
		// 256 is a multiple of 32, so the remainder is evenly spread.
		sb.WriteByte(enrolCodeAlphabet[int(v)%len(enrolCodeAlphabet)])
	}
	return sb.String(), nil
}

// normalizeEnrolCode makes what a person typed comparable: upper case, with
// spaces and dashes removed.
func normalizeEnrolCode(code string) string {
	return strings.NewReplacer(" ", "", "-", "").Replace(strings.ToUpper(code))
}
