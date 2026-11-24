// Package money provides cent-exact helpers for euro amounts.
package money

// Cents is an amount of money in hundredths of a euro. It marshals to JSON
// as a plain integer.
type Cents int64

// Split divides total into n parts that sum exactly to total. When the total
// does not divide evenly, the leftover cents are given one each to the first
// parts, so parts differ by at most one cent. It returns nil when n is not
// positive or total is negative.
func Split(total Cents, n int) []Cents {
	if n <= 0 || total < 0 {
		return nil
	}
	base := total / Cents(n)
	remainder := int(total % Cents(n))

	parts := make([]Cents, n)
	for i := range parts {
		parts[i] = base
		if i < remainder {
			parts[i]++
		}
	}
	return parts
}
