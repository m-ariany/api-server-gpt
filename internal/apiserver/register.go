package apiserver

import "time"

const (
	HTTP_API_TIMEOUT = time.Minute * 1
	SHUTDOWN_TIMEOUT = time.Second * 30
	PORT             = 80
)
