// Package auth handles user authentication including login, logout,
// and session management.
package auth

import (
	"errors"
	"time"
)

// Errors the device operations report, so callers can tell them apart with
// errors.Is.
var (
	// ErrUnknownUser means no user has the given name.
	ErrUnknownUser = errors.New("unknown user")
	// ErrUserHasDevice means the person already has a device that is not
	// revoked; each person has one device for now.
	ErrUserHasDevice = errors.New("user already has a device")
	// ErrDeviceNameTaken means another device that is not revoked has the name.
	ErrDeviceNameTaken = errors.New("device name already in use")
	// ErrUnknownDevice means no device that is not revoked has the name.
	ErrUnknownDevice = errors.New("unknown device")
	// ErrInvalidEnrolCode means the enrolment code is wrong, expired or used.
	ErrInvalidEnrolCode = errors.New("invalid or expired enrolment code")
)

// User represents a user in the system.
type User struct {
	ID           int64
	Username     string
	PasswordHash string
}

// Device is a phone that writes data for one person. Its zero times mean the
// event has not happened.
type Device struct {
	ID       int64
	UserID   int64
	Username string
	Name     string
	// CreatedAt is when the enrolment code was made.
	CreatedAt time.Time
	// EnrolCodeExpiresAt is when the pending code stops working; zero once the
	// device is enrolled or revoked.
	EnrolCodeExpiresAt time.Time
	EnrolledAt         time.Time
	LastSeenAt         time.Time
	RevokedAt          time.Time
}

// Status is "revoked", "enrolled", "expired" (a code nobody used in time) or
// "waiting" (a code that can still be used), as of now.
func (d Device) Status(now time.Time) string {
	switch {
	case !d.RevokedAt.IsZero():
		return "revoked"
	case !d.EnrolledAt.IsZero():
		return "enrolled"
	case !now.Before(d.EnrolCodeExpiresAt):
		return "expired"
	default:
		return "waiting"
	}
}
