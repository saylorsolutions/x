package cli

// MustGet is used with a [pflag.FlagSet] getter to panic if the flag is not defined, or is not the right type.
// The developer usually knows whether a get call will fail, so this function makes it easier to avoid global flag state.
func MustGet[T any](val T, err error) T {
	if err != nil {
		panic(err)
	}
	return val
}
