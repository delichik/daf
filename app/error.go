package app

type FatalError struct {
	msg         string
	original    error
	fatalCauser error
}

func (e *FatalError) Error() string {
	return "[fatal] " + e.msg + ": " +
		e.fatalCauser.Error() +
		", original error: " + e.original.Error()
}

func (e *FatalError) Unwrap() error {
	return e.fatalCauser
}

func ErrRollbackConfigFailed(original error, fatalCauser error) error {
	return &FatalError{
		msg:         "rollback config failed",
		original:    original,
		fatalCauser: fatalCauser,
	}
}
