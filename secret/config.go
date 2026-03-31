package secret

import "time"

type Config struct {
	Address       string
	Token         string
	Namespace     string
	KVMount       string
	Insecure      bool
	CACertFile    string
	CAPath        string
	TLSServerName string
	Timeout       time.Duration
}
