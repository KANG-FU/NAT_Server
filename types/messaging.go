package types

import (
	"fmt"
)

// -----------------------------------------------------------------------------
// ChatMessage

// NewEmpty implements types.Message.
func (c ChatMessage) NewEmpty() Message {
	return &ChatMessage{}
}

// Name implements types.Message.
func (ChatMessage) Name() string {
	return "chat"
}

// String implements types.Message.
func (c ChatMessage) String() string {
	return fmt.Sprintf("<%s>", c.Message)
}

// HTML implements types.Message.
func (c ChatMessage) HTML() string {
	return c.String()
}

// -----------------------------------------------------------------------------
// RegistrationMessage

// NewEmpty implements types.Message.
func (r RegistrationMessage) NewEmpty() Message {
	return &RegistrationMessage{}
}

// Name implements types.Message.
func (RegistrationMessage) Name() string {
	return "registration"
}

// String implements types.Message.
func (r RegistrationMessage) String() string {
	return fmt.Sprintf("<%s>", r.Message)
}

// HTML implements types.Message.
func (r RegistrationMessage) HTML() string {
	return r.String()
}
