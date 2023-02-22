package cookiejar

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/bool64/ctxd"
	"github.com/spf13/afero"
)

const (
	permReadonly = os.FileMode(0o600)
)

// jar is just for generating mock.
//
//go:generate mockery --name jar --exported --output mock --outpkg mock --filename jar.go
type jar interface {
	http.CookieJar
}

var _ http.CookieJar = (*PersistentJar)(nil)

// PersistentJar persists cookies to a file.
type PersistentJar struct {
	jar    *Jar
	fs     afero.Fs
	serder EntrySerDer
	logger ctxd.Logger

	autoSync bool
	filePath string
	filePerm os.FileMode

	lazyLoad sync.Once
}

// SetCookies implements the SetCookies method of the http.CookieJar interface.
//
// It does nothing if the URL's scheme is not HTTP or HTTPS.
func (j *PersistentJar) SetCookies(u *url.URL, cookies []*http.Cookie) {
	j.lazyLoad.Do(j.load)
	j.jar.SetCookies(u, cookies)

	if j.autoSync {
		if err := j.Sync(); err != nil {
			j.logger.Error(context.Background(), err.Error())
		}
	}
}

// Cookies implements the Cookies method of the http.CookieJar interface.
//
// It returns an empty slice if the URL's scheme is not HTTP or HTTPS.
func (j *PersistentJar) Cookies(u *url.URL) []*http.Cookie {
	j.lazyLoad.Do(j.load)

	return j.jar.Cookies(u)
}

// Sync persists cookies to the file.
func (j *PersistentJar) Sync() error {
	j.jar.mu.Lock()
	defer j.jar.mu.Unlock()

	ctx := ctxd.AddFields(context.Background(), "cookies.file", j.filePath)

	f, err := j.fs.OpenFile(filepath.Clean(j.filePath), os.O_RDWR|os.O_CREATE|os.O_TRUNC, j.filePerm)
	if err != nil {
		return ctxd.WrapError(ctx, err, "could not open file for persisting cookies")
	}

	defer func() {
		_ = f.Close() //nolint: errcheck
	}()

	if err := j.serder.Serialize(f, mapToExport(j.jar.entries)); err != nil {
		return ctxd.WrapError(ctx, err, "could not serialize cookies")
	}

	if err := f.Sync(); err != nil {
		return ctxd.WrapError(ctx, err, "could not sync cookies file")
	}

	return nil
}

func (j *PersistentJar) load() {
	j.jar.mu.Lock()
	defer j.jar.mu.Unlock()

	ctx := ctxd.AddFields(context.Background(), "cookies.file", j.filePath)

	f, err := j.fs.Open(filepath.Clean(j.filePath))
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			j.logger.Error(ctx, "could not open file for loading cookies", "error", err)
		}

		return
	}

	defer func() {
		_ = f.Close() //nolint: errcheck
	}()

	entries, err := j.serder.Deserialize(f)
	if err != nil {
		j.logger.Error(ctx, "could not deserialize cookies", "error", err)

		return
	}

	j.jar.entries, j.jar.nextSeqNum = mapToImport(entries)
}

// NewPersistentJar creates new persistent cookie jar.
func NewPersistentJar(opts ...PersistentJarOption) *PersistentJar {
	jar, _ := New(nil) //nolint: errcheck

	j := &PersistentJar{
		jar:      jar,
		fs:       afero.NewOsFs(),
		serder:   jsonSerDer{},
		logger:   ctxd.NoOpLogger{},
		autoSync: false,
		filePath: "cookies.json",
		filePerm: permReadonly,
	}

	for _, opt := range opts {
		opt.applyPersistentJarOption(j)
	}

	return j
}

// PersistentJarOption is an option to configure PersistentJar.
type PersistentJarOption interface {
	applyPersistentJarOption(j *PersistentJar)
}

type persistentJarOptionFunc func(j *PersistentJar)

func (f persistentJarOptionFunc) applyPersistentJarOption(j *PersistentJar) {
	f(j)
}

// WithFs sets the file system.
func WithFs(fs afero.Fs) PersistentJarOption {
	return persistentJarOptionFunc(func(j *PersistentJar) {
		j.fs = fs
	})
}

// WithSerDer sets the serializer/deserializer.
func WithSerDer(serder EntrySerDer) PersistentJarOption {
	return persistentJarOptionFunc(func(j *PersistentJar) {
		j.serder = serder
	})
}

// WithAutoSync sets the auto sync mode.
func WithAutoSync(autoSync bool) PersistentJarOption {
	return persistentJarOptionFunc(func(j *PersistentJar) {
		j.autoSync = autoSync
	})
}

// WithFilePath sets the file path.
func WithFilePath(filePath string) PersistentJarOption {
	return persistentJarOptionFunc(func(j *PersistentJar) {
		j.filePath = filePath
	})
}

// WithFilePerm sets the file permission.
func WithFilePerm(filePerm os.FileMode) PersistentJarOption {
	return persistentJarOptionFunc(func(j *PersistentJar) {
		j.filePerm = filePerm
	})
}

// WithLogger sets the logger.
func WithLogger(logger ctxd.Logger) PersistentJarOption {
	return persistentJarOptionFunc(func(j *PersistentJar) {
		j.logger = logger
	})
}

// WithPublicSuffixList sets the public suffix list.
func WithPublicSuffixList(list PublicSuffixList) PersistentJarOption {
	return persistentJarOptionFunc(func(j *PersistentJar) {
		j.jar.psList = list
	})
}

// Entry is a public presentation of the entry struct.
type Entry struct {
	Name       string
	Value      string
	Domain     string
	Path       string
	SameSite   string
	Secure     bool
	HttpOnly   bool
	Persistent bool
	HostOnly   bool
	Expires    time.Time
	Creation   time.Time
	LastAccess time.Time
	SeqNum     uint64
}

func mapToExport(entries map[string]map[string]entry) map[string]map[string]Entry {
	exported := make(map[string]map[string]Entry)

	for domain, domainCookies := range entries {
		exported[domain] = make(map[string]Entry)

		for name, cookie := range domainCookies {
			exported[domain][name] = Entry{
				Name:       cookie.Name,
				Value:      cookie.Value,
				Domain:     cookie.Domain,
				Path:       cookie.Path,
				SameSite:   cookie.SameSite,
				Secure:     cookie.Secure,
				HttpOnly:   cookie.HttpOnly,
				Persistent: cookie.Persistent,
				HostOnly:   cookie.HostOnly,
				Expires:    cookie.Expires,
				Creation:   cookie.Creation,
				LastAccess: cookie.LastAccess,
				SeqNum:     cookie.seqNum,
			}
		}
	}

	return exported
}

func mapToImport(entries map[string]map[string]Entry) (map[string]map[string]entry, uint64) {
	nextSeqNum := uint64(0)
	imported := make(map[string]map[string]entry)

	for domain, domainCookies := range entries {
		imported[domain] = make(map[string]entry)

		for name, cookie := range domainCookies {
			imported[domain][name] = entry{
				Name:       cookie.Name,
				Value:      cookie.Value,
				Domain:     cookie.Domain,
				Path:       cookie.Path,
				SameSite:   cookie.SameSite,
				Secure:     cookie.Secure,
				HttpOnly:   cookie.HttpOnly,
				Persistent: cookie.Persistent,
				HostOnly:   cookie.HostOnly,
				Expires:    cookie.Expires,
				Creation:   cookie.Creation,
				LastAccess: cookie.LastAccess,
				seqNum:     cookie.SeqNum,
			}

			if cookie.SeqNum > nextSeqNum {
				nextSeqNum = cookie.SeqNum + 1
			}
		}
	}

	return imported, nextSeqNum
}

// EntrySerDer is an interface for serializing and deserializing entries.
type EntrySerDer interface {
	Serialize(w io.Writer, entries map[string]map[string]Entry) error
	Deserialize(r io.Reader) (map[string]map[string]Entry, error)
}

// jsonSerDer is a JSON serializer and deserializer.
type jsonSerDer struct{}

func (jsonSerDer) Serialize(w io.Writer, entries map[string]map[string]Entry) error {
	return json.NewEncoder(w).Encode(entries)
}

func (jsonSerDer) Deserialize(r io.Reader) (map[string]map[string]Entry, error) {
	var entries map[string]map[string]Entry

	if err := json.NewDecoder(r).Decode(&entries); err != nil {
		return nil, err
	}

	return entries, nil
}
