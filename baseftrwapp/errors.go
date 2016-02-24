package baseftrwapp

// InvalidRequestError for if a bad request has been issued to a method
type invalidRequestError interface {
	InvalidRequestDetails() string
}
