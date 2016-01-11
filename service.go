package baseftrwapp

import (
	"encoding/json"

	"github.com/Financial-Times/go-fthealth/v1a"
)

// Service defines the functions any read-write application needs to implement
type Service interface {
	Write(thing interface{}) error
	Read(uuid string) (thing interface{}, found bool, err error)
	Delete(uuid string) (found bool, err error)
	DecodeJSON(*json.Decoder) (thing interface{}, identity string, err error)
	Count() (int, error)
	Check() (check v1a.Check)
}
