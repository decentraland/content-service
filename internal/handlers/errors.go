package handlers

type InvalidArgument struct {
	Message string
}

func (e InvalidArgument) Error() string {
	return e.Message
}

type RequiredValueError struct {
	Message string
}

func (e RequiredValueError) Error() string {
	return e.Message
}

type UnexpectedError struct {
	Message string
	error
}

type UnauthorizedError struct {
	Message string
}

func (e UnauthorizedError) Error() string {
	return e.Message
}
