# REST Response

Base Response for REST API call as follows:

```go
type JSONResponse struct {
    Data    interface{}    `json:"data,omitempty"`
    Code    int            `json:"code,omitempty"`
    Message string         `json:"message,omitempty"`
    Error   *ErrorResponse `json:"error,omitempty"`
}

type ErrorResponse struct {
    ErrorCode    int    `json:"error_code,omitempty"`
    ErrorMessage string `json:"error_message,omitempty"`
}
```

### Explanation

- Data is for returning data, you can set it with any type or struct value
- Code is for HTTP code
- Error is using when you REST have some issue (might be from client or server)
- ErrorCode inside ErrorResponse is used to define your error code. This make easier to debug or check the error.
- Message inside ErrorResponse is your message that will be shown to client.

### Example
Sample success response
```json
{
  "data": {
    "id": 1,
    "name": "John"
  },
  "code": 200
}
```

Sample error response
```json
{
  "code": 400,
  "error": {
    "error_code": 123,
    "message": "Invalid user action"
  }
}
```

### Usage

This response is intended to use inside controller as follows
```go
return rest.NewJSONResponse().setData(data)
```

And if you need returning error you can do
```go
return rest.NewJSONResponse().setError(err)
```


This response provide predefined error response for common error with its http code.

Maybe, in your usecase you need other error response to handle. So, you may need to extend this response and add custom response on your repository.  Will provide an example later.