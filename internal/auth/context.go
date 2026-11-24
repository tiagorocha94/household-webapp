package auth

import "context"

const (
	userContextKey   contextKey = "user"
	deviceContextKey contextKey = "device"
)

type contextKey string

// WithUser returns a context that carries the authenticated user.
func WithUser(ctx context.Context, user *User) context.Context {
	return context.WithValue(ctx, userContextKey, user)
}

// UserFromContext retrieves the authenticated User from the request context.
// Returns nil if no user is present.
func UserFromContext(ctx context.Context) *User {
	user, _ := ctx.Value(userContextKey).(*User)
	return user
}

// WithDevice returns a context that carries the device a request was
// authenticated as. WithUser carries the device's person.
func WithDevice(ctx context.Context, device *Device) context.Context {
	return context.WithValue(ctx, deviceContextKey, device)
}

// DeviceFromContext retrieves the authenticated Device from the request
// context. Returns nil if the request was not made by a device.
func DeviceFromContext(ctx context.Context) *Device {
	device, _ := ctx.Value(deviceContextKey).(*Device)
	return device
}
