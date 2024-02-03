// Code generated by protoc-gen-validate. DO NOT EDIT.
// source: consensus/mbft/proto/transport.proto

package proto

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"net/mail"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"google.golang.org/protobuf/types/known/anypb"
)

// ensure the imports are used
var (
	_ = bytes.MinRead
	_ = errors.New("")
	_ = fmt.Print
	_ = utf8.UTFMax
	_ = (*regexp.Regexp)(nil)
	_ = (*strings.Reader)(nil)
	_ = net.IPv4len
	_ = time.Duration(0)
	_ = (*url.URL)(nil)
	_ = (*mail.Address)(nil)
	_ = anypb.Any{}
	_ = sort.Sort
)

// Validate checks the field values on TransportMessage with the rules defined
// in the proto definition for this message. If any rules are violated, the
// first error encountered is returned, or nil if there are no violations.
func (m *TransportMessage) Validate() error {
	return m.validate(false)
}

// ValidateAll checks the field values on TransportMessage with the rules
// defined in the proto definition for this message. If any rules are
// violated, the result is a list of violation errors wrapped in
// TransportMessageMultiError, or nil if none found.
func (m *TransportMessage) ValidateAll() error {
	return m.validate(true)
}

func (m *TransportMessage) validate(all bool) error {
	if m == nil {
		return nil
	}

	var errors []error

	// no validation rules for Data

	if len(errors) > 0 {
		return TransportMessageMultiError(errors)
	}

	return nil
}

// TransportMessageMultiError is an error wrapping multiple validation errors
// returned by TransportMessage.ValidateAll() if the designated constraints
// aren't met.
type TransportMessageMultiError []error

// Error returns a concatenation of all the error messages it wraps.
func (m TransportMessageMultiError) Error() string {
	var msgs []string
	for _, err := range m {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// AllErrors returns a list of validation violation errors.
func (m TransportMessageMultiError) AllErrors() []error { return m }

// TransportMessageValidationError is the validation error returned by
// TransportMessage.Validate if the designated constraints aren't met.
type TransportMessageValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e TransportMessageValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e TransportMessageValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e TransportMessageValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e TransportMessageValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e TransportMessageValidationError) ErrorName() string { return "TransportMessageValidationError" }

// Error satisfies the builtin error interface
func (e TransportMessageValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sTransportMessage.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = TransportMessageValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = TransportMessageValidationError{}
