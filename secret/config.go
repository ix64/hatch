package secret

import "time"

type Config struct {
	Address        string
	Token          string
	Namespace      string
	KVMount        string
	Insecure       bool
	CACertFile     string
	CAPath         string
	TLSServerName  string
	TimeoutSeconds int
}

func (c Config) Timeout() time.Duration {
	return time.Duration(c.TimeoutSeconds) * time.Second
}
