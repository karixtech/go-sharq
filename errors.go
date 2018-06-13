package sharq

type ProxySharqError struct {
	Message string
}

func (p ProxySharqError) Error() string {
	return p.Message
}
