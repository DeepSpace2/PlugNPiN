package errors

type (
	InvalidSchemeError struct {
		Msg string
	}
	MalformedIPLabelError struct {
		Msg string
	}
	NonExistingLabelsError struct {
		Msg string
	}
)

func (e *InvalidSchemeError) Error() string {
	return e.Msg
}

func (e *MalformedIPLabelError) Error() string {
	return e.Msg
}

func (e *NonExistingLabelsError) Error() string {
	return e.Msg
}
