package helpers

import (
	"os"
	"strings"
)

func CheckDomainError(url string) bool {
	if url == os.Getenv("DOMAIN") {
		return false
	}

	newUrl := strings.Replace(url, "https://", "", 1)
	newUrl = strings.Replace(newUrl, "http://", "", 1)
	newUrl = strings.Replace(newUrl, "www.", "", 1)
	newUrl = strings.Split(newUrl, "/")[0]

	if newUrl == os.Getenv("DOMAIN") {
		return false
	}

	return true

}

func EnforceHTTP(url string) string {
	if url[0:4] != "http" {
		return "http://" + url
	}

	return url
}
