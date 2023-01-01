// Package output registers different output methods for dnsmonster.
// each output will register itself by running the init function.
// in the main package, if the output type is zero, the output will automatically
// be de-registered from dispatch. each output can provide its own specific flags
// and also benefit from generalflags using the `util` package
package output
