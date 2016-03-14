package baseftrwapp

// InvalidRequestError for if a bad request has been issued to a method
type invalidRequestError interface {
	InvalidRequestDetails() string
}

// NoContentReturnedError if No Content is returned for the request
type NoContentReturnedError interface {
	NoContentReturnedDetails() string
}
