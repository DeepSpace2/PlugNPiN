package errors

type (
	NonExistingLabelsError struct {
		Msg string
	}
	MalformedIPLabelError struct {
		Msg string
	}
)

func (e *NonExistingLabelsError) Error() string {
	return e.Msg
}

func (e *MalformedIPLabelError) Error() string {
	return e.Msg
}
