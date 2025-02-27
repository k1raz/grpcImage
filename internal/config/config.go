package config

type Config struct {
	ServerAddress       string
	StorageDir          string
	UploadDownloadLimit int
	ListLimit           int
}

func NewDefaultConfig() *Config {
	return &Config{
		ServerAddress:       ":50051",
		StorageDir:          "./storage",
		UploadDownloadLimit: 10,
		ListLimit:           100,
	}
}
