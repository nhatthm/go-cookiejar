// Package mock provides mocks for the cookiejar package.
package mock

import (
	"net/http"
	"testing"
)

// Mocker is Jar mocker.
type Mocker func(tb testing.TB) *Jar

// NopJar is no mock Jar.
var NopJar = Mock()

var _ http.CookieJar = (*Jar)(nil)

// Mock creates Jar mock with cleanup to ensure all the expectations are met.
func Mock(mocks ...func(j *Jar)) Mocker {
	return func(tb testing.TB) *Jar {
		tb.Helper()

		j := NewJar(tb)

		for _, mock := range mocks {
			mock(j)
		}

		return j
	}
}
