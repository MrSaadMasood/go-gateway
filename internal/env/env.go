package env

import (
	"os"
)

func GetDbConnString() string {
	return os.Getenv("DATABASE_URL")
}
