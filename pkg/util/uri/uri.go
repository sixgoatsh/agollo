package uri

import "strings"

func NormalizeURL(url string) string {
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = "http://" + url
	}

	return strings.TrimSuffix(url, "/")
}

func SplitCommaSeparatedURL(s string) []string {
	var urls []string
	for _, url := range strings.Split(s, ",") {
		urls = append(urls, NormalizeURL(strings.TrimSpace(url)))
	}

	return urls
}
