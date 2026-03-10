// Package clock contains the Clock interface, an abstraction over the functions in the standard time package.
// A trivial implementation of the Clock interface using the time package is provided by Real.
// Other packages can provide alternative implementations of the Clock interface that behave differently,
// for example to implement simulations at arbitrary speeds.
package clock
