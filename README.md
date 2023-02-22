# Cookiejar

[![GitHub Releases](https://img.shields.io/github/v/release/nhatthm/go-cookiejar)](https://github.com/nhatthm/go-cookiejar/releases/latest)
[![Build Status](https://github.com/nhatthm/go-cookiejar/actions/workflows/test.yaml/badge.svg)](https://github.com/nhatthm/go-cookiejar/actions/workflows/test.yaml)
[![codecov](https://codecov.io/gh/nhatthm/go-cookiejar/branch/master/graph/badge.svg?token=eTdAgDE2vR)](https://codecov.io/gh/nhatthm/go-cookiejar)
[![Go Report Card](https://goreportcard.com/badge/go.nhat.io/cookiejar)](https://goreportcard.com/report/go.nhat.io/cookiejar)
[![GoDevDoc](https://img.shields.io/badge/dev-doc-00ADD8?logo=go)](https://pkg.go.dev/go.nhat.io/cookiejar)
[![Donate](https://img.shields.io/badge/Donate-PayPal-green.svg)](https://www.paypal.com/donate/?hosted_button_id=PJZSGJN57TDJY)

The Persistent Cookiejar is a fork of [`net/http/cookiejar`](https://pkg.go.dev/net/http/cookiejar) which also implements methods for persisting the cookies to
a filesystem and retrieving them using [`spf13/afero`](https://github.com/spf13/afero)

## Prerequisites

- `Go >= 1.20`

## Install

```bash
go get go.nhat.io/cookiejar
```

## Usage

Construct the cookiejar with the following options:

| Option                 | Description                                                                                                                         |   Default Value   |
|:-----------------------|:------------------------------------------------------------------------------------------------------------------------------------|:-----------------:|
| `WithFilePath`         | The path to the file to store the cookies                                                                                           | `"cookies.json"`  |
| `WithFilePerm`         | The file permission to use for persisting the cookies                                                                               |      `0600`       |
| `WithAutoSync`         | Whether to automatically sync the cookies to the file after each request                                                            |      `false`      |
| `WithLogger`           | The logger to use for logging                                                                                                       |      No log       |
| `WithFs`               | The filesystem to use for persisting the cookies                                                                                    | `afero.NewOsFs()` |
| `WithSerDer`           | The serializer/deserializer to use for persisting the cookies                                                                       |      `json`       |
| `WithPublicSuffixList` | The public suffix list to use for cookie domain matching </br> All users of cookiejar should import `golang.org/x/net/publicsuffix` |       `nil`       |

Example:

```go
package example

import (
	"net/http"

	"go.nhat.io/cookiejar"
)

func newClient() *http.Client {
	jar := cookiejar.NewPersistentJar(
		cookiejar.WithFilePath("/path/to/cookies.json"),
		cookiejar.WithFilePerm(0755),
		cookiejar.WithAutoSync(true),
	)

	return &http.Client{
		Jar: jar,
	}
}

```

## Examples

```go
package cookiejar_test

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"

	"go.nhat.io/cookiejar"
)

func ExampleNewPersistentJar() {
	tempDir, err := os.MkdirTemp(os.TempDir(), "example")
	if err != nil {
		log.Fatal(err)
	}

	defer os.RemoveAll(tempDir)

	cookiesFile := filepath.Join(tempDir, "cookies")

	// Start a server to give us cookies.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if cookie, err := r.Cookie("Flavor"); err != nil {
			http.SetCookie(w, &http.Cookie{Name: "Flavor", Value: "Chocolate Chip"})
		} else {
			cookie.Value = "Oatmeal Raisin"
			http.SetCookie(w, cookie)
		}
	}))
	defer ts.Close()

	u, err := url.Parse(ts.URL)
	if err != nil {
		log.Fatal(err)
	}

	jar := cookiejar.NewPersistentJar(
		cookiejar.WithFilePath(cookiesFile),
		cookiejar.WithAutoSync(true),
		// All users of cookiejar should import "golang.org/x/net/publicsuffix"
		cookiejar.WithPublicSuffixList(publicsuffix.List),
	)

	client := &http.Client{
		Jar: jar,
	}

	if _, err = client.Get(u.String()); err != nil {
		log.Fatal(err)
	}

	fmt.Println("After 1st request:")
	for _, cookie := range jar.Cookies(u) {
		fmt.Printf("  %s: %s\n", cookie.Name, cookie.Value)
	}

	if _, err = client.Get(u.String()); err != nil {
		log.Fatal(err)
	}

	fmt.Println("After 2nd request:")
	for _, cookie := range jar.Cookies(u) {
		fmt.Printf("  %s: %s\n", cookie.Name, cookie.Value)
	}

	// Output:
	// After 1st request:
	//   Flavor: Chocolate Chip
	// After 2nd request:
	//   Flavor: Oatmeal Raisin
}
```

## Donation

If this project help you reduce time to develop, you can give me a cup of coffee :)

### Paypal donation

[![paypal](https://www.paypalobjects.com/en_US/i/btn/btn_donateCC_LG.gif)](https://www.paypal.com/donate/?hosted_button_id=PJZSGJN57TDJY)

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;or scan this

<img src="https://user-images.githubusercontent.com/1154587/113494222-ad8cb200-94e6-11eb-9ef3-eb883ada222a.png" width="147px" />
