package services

type Service struct {
	CurrentStep  string
	RateLimit    int
	BlockedPaths []string
	IsAccessible bool
}
