package shared

import "os"

type Settings struct {
	SignalServerUrl string
}

func GetSettings() Settings {
	return Settings{
		SignalServerUrl: os.Getenv("SIGNAL_SERVER_URL"),
	}
}
