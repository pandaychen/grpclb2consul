package utils

import (
	//"string"
	"fmt"
)

func GetGrpcScheme(scheme string) string {
	return fmt.Sprintf("%s:///", scheme)
}
