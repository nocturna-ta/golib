package custerr

import "fmt"

type ErrChain struct {
	Message string
	Cause   error
	Code    int
	Fields  map[string]any
	Type    error
}

func (err ErrChain) Error() string {
	bcoz := ""
	fields := ""
	if err.Cause != nil {
		bcoz = fmt.Sprint(" because {", err.Cause.Error(), "}")
		if len(err.Fields) > 0 {
			fields = fmt.Sprintf(" with Fields {%+v}", err.Fields)
		}
	}
	return fmt.Sprint(err.Message, bcoz, fields)
}
