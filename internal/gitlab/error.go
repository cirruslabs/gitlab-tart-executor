package gitlab

type SystemFailureError struct {
	inner error
}

func NewSystemFailureError(err error) error {
	return &SystemFailureError{
		inner: err,
	}
}

func (outer *SystemFailureError) Error() string {
	return outer.inner.Error()
}

func (outer *SystemFailureError) Unwrap() error {
	return outer.inner
}
