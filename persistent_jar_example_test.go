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
