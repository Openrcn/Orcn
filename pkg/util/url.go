package util

import (
	"net/url"
)

func URLQueryEscape(s string) string {
	return url.QueryEscape(s)
}