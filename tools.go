// Package toolkit provides utility functions for the application.
package toolkit

// Import the cryptographically secure random number generator package.
import "crypto/rand"

// Define the alphabet/source characters from which the random string will be generated.
// It contains 64 characters: a-z, A-Z, 0-9, and the characters '_' and '+'.
const randomStringSource = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_+"

// Think about struct as about class, small classes.
// Tools is an empty struct used as a receiver to group utility methods.
type Tools struct{}

// RandomString generates a cryptographically secure random string of length n.
func (t *Tools) RandomString(n int) string {
	// Initialize a slice of runes 's' of size 'n' to hold the generated string characters.
	// Convert the 'randomStringSource' string to a slice of runes 'r' for efficient indexing.
	s, r := make([]rune, n), []rune(randomStringSource)

	// Loop 'n' times to generate each character of the random string.
	for i := range s {
		// Generate a cryptographically secure random prime number 'p' of bit-length equal to len(r) (64 bits).
		// rand.Reader is the global, shared CSPRNG source.
		p, _ := rand.Prime(rand.Reader, len(r))

		// Convert the random prime number 'p' (big.Int) to a uint64 variable 'x'.
		// Convert the length of the rune slice 'r' (64) to a uint64 variable 'y'.
		x, y := p.Uint64(), uint64(len(r))

		// Use the modulo operator (x % y) to get a random index in the range [0, 63].
		// Assign the character from 'r' at that index to the current position in the slice 's'.
		s[i] = r[x%y]
	}

	// Convert the final slice of runes 's' back into a string and return it.
	return string(s)
}
