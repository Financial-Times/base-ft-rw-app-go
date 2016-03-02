package baseftrwapp

import (
	"encoding/json"
	"fmt"
)

// Service defines the functions any read-write application needs to implement
type Service interface {
	Write(thing interface{}) error
	Read(uuid string) (thing interface{}, found bool, err error)
	Delete(uuid string) (found bool, err error)
	DecodeJSON(*json.Decoder) (thing interface{}, identity string, err error)
	Count() (int, error)
	Check() error
	Initialise() error
}

// ServiceError is a possible error the Service can return
type ConflictError struct {
	msg string
	err error
}

func (err ConflictError) Error() string {
	return fmt.Sprintf("%s - %s", err.msg, err.err.Error())
}

func NewConflictError(msg string, err error) error {
	ce := new(ConflictError)
	ce.msg = msg
	ce.err = err
	return ce
}
