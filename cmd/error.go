package cmd

func Wrap(err error) *GoreleaserError {
	return WrapWithCode(err, 1)
}

func WrapWithCode(err error, code int) *GoreleaserError {
	if err == nil {
		return nil
	}
	return &GoreleaserError{
		err:  err,
		exit: code,
	}
}

type GoreleaserError struct {
	err  error
	exit int
}

func (e *GoreleaserError) Error() string {
	return e.err.Error()
}

func (e *GoreleaserError) Exit() int {
	return e.exit
}
