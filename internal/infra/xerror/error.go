package xerror

type CustomError struct {
    Code    int `json:"-"`
    Message string `json:"message"`
    Err error `json:"error"`
}

func NewCustomError(code int, message string, err error) *CustomError {
    return &CustomError{
        Code:    code,
        Message: message,
        Err: err,
    }
}

func (e *CustomError) Error() string {
    return e.Message
}