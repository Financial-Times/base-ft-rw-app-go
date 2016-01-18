package baseftrwapp

import (
	"encoding/json"
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
