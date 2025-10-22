package regexpx

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var (
	spacesPattern   = regexp.MustCompile(`\s+`)
	namePattern     = regexp.MustCompile(`^[a-z]+$`)
	targetPattern   = regexp.MustCompile(`\[\[:\$([a-z]+):]]`)
	ErrInvalidName  = errors.New("invalid pattern name")
	ErrUndefinedRef = errors.New("undefined reference")
)

// VarPattern allows defining and reusing regex patterns by overloading the character class semantics in regex patterns to act as a replacement target.
// Note that this is likely only useful to make patterns DRY in more complicated regex use-cases.
//
// The normal syntax for a character class is [[:name:]], and can be negated like [^[:name:]] or [[:^name]], where "name" is the character class name.
// These names always use letters. See [regexp/syntax] for details on the stdlib pattern semantics.
//
// This type uses a pattern like this to express a pattern replacement target:
//
//	[[:$pattern:]]
//
// VarPattern will parse this out of the pattern string and replace it with a known pattern named "pattern."
// To keep things simple, only lowercase letters may be used to define a pattern. No numbers, symbols, or spaces.
// This is to keep pattern replacement unambiguous and predictable, considering how many semantic characters there can be in regular expressions.
type VarPattern struct {
	values map[string]string
}

func NewVarPattern() *VarPattern {
	return &VarPattern{
		values: map[string]string{},
	}
}

// Define is used to define a named pattern that will be replaced with calls to Compile.
func (v *VarPattern) Define(name, pattern string) error {
	if spacesPattern.MatchString(name) {
		return fmt.Errorf("%w: spaces are not allowed in pattern names", ErrInvalidName)
	}
	if !namePattern.MatchString(name) {
		return fmt.Errorf("%w: '%s'", ErrInvalidName, name)
	}
	if _, err := regexp.Compile(pattern); err != nil {
		return err
	}
	pattern, err := v.replacePattern(pattern)
	if err != nil {
		return err
	}
	v.values[name] = pattern
	return nil
}

func (v *VarPattern) MustDefine(name, pattern string) {
	if err := v.Define(name, pattern); err != nil {
		panic(err)
	}
}

func (v *VarPattern) Compile(pattern string) (*regexp.Regexp, error) {
	pat, err := v.replacePattern(pattern)
	if err != nil {
		return nil, err
	}
	return regexp.Compile(pat)
}

func (v *VarPattern) MustCompile(pattern string) *regexp.Regexp {
	pat, err := v.Compile(pattern)
	if err != nil {
		panic(err)
	}
	return pat
}

func (v *VarPattern) replacePattern(pattern string) (string, error) {
	bs := []byte(pattern)
	matches := targetPattern.FindAllSubmatchIndex(bs, -1)
	if len(matches) == 0 {
		return pattern, nil
	}
	var (
		buf      strings.Builder
		lastStop int
	)
	for _, match := range matches {
		// If a match is found, then it must have a name to lookup.
		name := string(bs[match[2]:match[3]])
		replPattern, ok := v.values[name]
		if !ok {
			return "", fmt.Errorf("%w '%s' in pattern '%s'", ErrUndefinedRef, name, pattern)
		}
		buf.Write(bs[lastStop:match[0]])
		lastStop = match[1]
		buf.WriteString(replPattern)
	}
	buf.Write(bs[lastStop:])
	newPattern := buf.String()
	if _, err := regexp.Compile(newPattern); err != nil {
		return "", fmt.Errorf("unable to compile pattern with replacements '%s': %w", newPattern, err)
	}
	return buf.String(), nil
}
