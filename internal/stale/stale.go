package stale

import "errors"

type Error struct{ Msg string }

func (e Error) Error() string {
	if e.Msg != "" {
		return e.Msg
	}
	return "page changed since the decision. Observe again"
}

func (e Error) Is(target error) bool {
	_, ok := target.(Error)
	return ok
}

func Is(err error) bool {
	return errors.Is(err, Error{})
}
