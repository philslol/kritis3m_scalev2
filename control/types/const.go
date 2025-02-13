package types

import "time"

const (
	HTTPReadTimeout     = 30 * time.Second
	HTTPShutdownTimeout = 3 * time.Second

	JSONLogFormat = "json"
	TextLogFormat = "text"

	KeepAliveInterval = 60 * time.Second
	MaxHostnameLength = 255
	//timestamp format
	TimestampFormat = time.RFC3339
)
