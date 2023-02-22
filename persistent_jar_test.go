package cookiejar_test

import (
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"testing"

	"github.com/bool64/ctxd"
	"github.com/spf13/afero"
	"github.com/spf13/afero/mem"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/swaggest/assertjson"
	"go.nhat.io/aferomock"

	"go.nhat.io/cookiejar"
)

func TestPersistentJar_SetCookies_NoAutoSync_Success(t *testing.T) {
	t.Parallel()

	const filePath = "/tmp/cookies.json"

	const fileContent = `{
  "example.com": {
    "example.com;/;id": {
      "Name": "id",
      "Value": "40",
      "Domain": "example.com",
      "Path": "/",
      "SeqNum": 0
    },
    "example.com;/;username": {
      "Name": "username",
      "Value": "john",
      "Domain": "example.com",
      "Path": "/",
      "SeqNum": 1
    }
  }
}`

	fileData := mem.CreateFile("test")

	f := mem.NewFileHandle(fileData)
	_, _ = f.WriteString(fileContent) //nolint: errcheck
	_ = f.Close()                     //nolint: errcheck

	fs := aferomock.MockFs(func(fs *aferomock.Fs) {
		fs.On("Open", filePath).Once().
			Return(mem.NewFileHandle(fileData), nil)

		fs.On("OpenFile", filePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, os.FileMode(0o755)).Once().
			Return(mem.NewFileHandle(fileData), nil)
	})(t)

	j := cookiejar.NewPersistentJar(
		cookiejar.WithAutoSync(false),
		cookiejar.WithFs(fs),
		cookiejar.WithFilePath(filePath),
		cookiejar.WithFilePerm(0o755),
		cookiejar.WithLogger(ctxd.NoOpLogger{}),
	)

	u := &url.URL{Scheme: "https", Host: "example.com"}

	j.SetCookies(u, []*http.Cookie{{
		Name:  "id",
		Value: "42",
	}, {
		Name:  "email",
		Value: "john@example.com",
	}})

	// File is not synced, so it should not contain new cookie.
	assertFileDataEqual(t, fileContent, fileData)

	err := j.Sync()
	require.NoError(t, err)

	// Cookie is synced, so it should contain new cookie.
	actualCookies := j.Cookies(u)
	expectedCookies := []*http.Cookie{{
		Name:  "id",
		Value: "42",
	}, {
		Name:  "username",
		Value: "john",
	}, {
		Name:  "email",
		Value: "john@example.com",
	}}

	assert.Equal(t, expectedCookies, actualCookies)

	// File is synced, so it should contain new cookie.
	expectedContent := `{
  "example.com": {
    "example.com;/;id": {
      "Name": "id",
      "Value": "42",
      "Domain": "example.com",
      "Path": "/",
      "SameSite": "",
      "Secure": false,
      "HttpOnly": false,
      "Persistent": false,
      "HostOnly": true,
      "Expires": "9999-12-31T23:59:59Z",
      "Creation": "<ignore-diff>",
      "LastAccess": "<ignore-diff>",
      "SeqNum": 0
    },
    "example.com;/;username": {
      "Name": "username",
      "Value": "john",
      "Domain": "example.com",
      "Path": "/",
      "SameSite": "",
      "Secure": false,
      "HttpOnly": false,
      "Persistent": false,
      "HostOnly": false,
      "Expires": "0001-01-01T00:00:00Z",
      "Creation": "<ignore-diff>",
      "LastAccess": "<ignore-diff>",
      "SeqNum": 1
    },
    "example.com;/;email": {
      "Name": "email",
      "Value": "john@example.com",
      "Domain": "example.com",
      "Path": "/",
      "SameSite": "",
      "Secure": false,
      "HttpOnly": false,
      "Persistent": false,
      "HostOnly": true,
      "Expires": "9999-12-31T23:59:59Z",
      "Creation": "<ignore-diff>",
      "LastAccess": "<ignore-diff>",
      "SeqNum": 2
    }
  }
}`

	assertFileDataJSONEqual(t, expectedContent, fileData)
}

func TestPersistentJar_SetCookies_AutoSync_Error(t *testing.T) {
	t.Parallel()

	const filePath = "/tmp/cookies.json"

	testCases := []struct {
		scenario string
		mockFs   aferomock.FsMocker
	}{
		{
			scenario: "could not open file for writing",
			mockFs: aferomock.MockFs(func(fs *aferomock.Fs) {
				fs.On("Open", mock.Anything).Once().
					Return(nil, os.ErrNotExist)

				fs.On("OpenFile", mock.Anything, mock.Anything, mock.Anything).Once().
					Return(nil, errors.New("open file error"))
			}),
		},
		{
			scenario: "could not encode cookies",
			mockFs: aferomock.MockFs(func(fs *aferomock.Fs) {
				fs.On("Open", mock.Anything).Once().
					Return(nil, os.ErrNotExist)

				f := mem.NewFileHandle(mem.CreateFile("test"))
				_ = f.Close() //nolint: errcheck

				fs.On("OpenFile", mock.Anything, mock.Anything, mock.Anything).Once().
					Return(f, nil)
			}),
		},
		{
			scenario: "could not sync file",
			mockFs: aferomock.MockFs(func(fs *aferomock.Fs) {
				fs.On("Open", mock.Anything).Once().
					Return(nil, os.ErrNotExist)

				f := &fileWithSyncError{
					File:      mem.NewFileHandle(mem.CreateFile("test")),
					SyncError: errors.New("sync error"),
				}

				fs.On("OpenFile", mock.Anything, mock.Anything, mock.Anything).Once().
					Return(f, nil)
			}),
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.scenario, func(t *testing.T) {
			t.Parallel()

			j := cookiejar.NewPersistentJar(
				cookiejar.WithAutoSync(true),
				cookiejar.WithFs(tc.mockFs(t)),
				cookiejar.WithFilePath(filePath),
				cookiejar.WithLogger(ctxd.NoOpLogger{}),
			)

			u := &url.URL{Scheme: "https", Host: "example.com"}

			j.SetCookies(u, []*http.Cookie{{
				Name:  "username",
				Value: "john.doe",
			}})
		})
	}
}

func TestPersistentJar_SetCookies_AutoSync_Success(t *testing.T) {
	t.Parallel()

	const filePath = "/tmp/cookies.json"

	fileData := mem.CreateFile("test")

	fs := aferomock.MockFs(func(fs *aferomock.Fs) {
		fs.On("Open", filePath).Once().
			Return(nil, os.ErrNotExist)

		fs.On("OpenFile", filePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, os.FileMode(0o755)).Once().
			Return(mem.NewFileHandle(fileData), nil)
	})(t)

	j := cookiejar.NewPersistentJar(
		cookiejar.WithAutoSync(true),
		cookiejar.WithFs(fs),
		cookiejar.WithFilePath(filePath),
		cookiejar.WithFilePerm(0o755),
		cookiejar.WithLogger(ctxd.NoOpLogger{}),
	)

	u := &url.URL{Scheme: "https", Host: "example.com"}

	j.SetCookies(u, []*http.Cookie{{
		Name:  "username",
		Value: "john.doe",
	}})

	expectedContent := `{
  "example.com": {
    "example.com;/;username": {
      "Name": "username",
      "Value": "john.doe",
      "Domain": "example.com",
      "Path": "/",
      "SameSite": "",
      "Secure": false,
      "HttpOnly": false,
      "Persistent": false,
      "HostOnly": true,
      "Expires": "9999-12-31T23:59:59Z",
      "Creation": "<ignore-diff>",
      "LastAccess": "<ignore-diff>",
      "SeqNum": 0
    }
  }
}`

	assertFileDataJSONEqual(t, expectedContent, fileData)
}

func TestPersistentJar_Cookies(t *testing.T) {
	t.Parallel()

	const filePath = "/tmp/cookies.json"

	testCases := []struct {
		scenario string
		mockFs   aferomock.FsMocker
		expected []*http.Cookie
	}{
		{
			scenario: "file not found",
			mockFs: aferomock.MockFs(func(fs *aferomock.Fs) {
				fs.On("Open", mock.Anything).Once().
					Return(nil, os.ErrNotExist)
			}),
		},
		{
			scenario: "open file error",
			mockFs: aferomock.MockFs(func(fs *aferomock.Fs) {
				fs.On("Open", mock.Anything).Once().
					Return(nil, errors.New("open error"))
			}),
		},
		{
			scenario: "broken json",
			mockFs: aferomock.MockFs(func(fs *aferomock.Fs) {
				f := mem.NewFileHandle(mem.CreateFile("cookies.json"))
				_, _ = f.WriteString("{")      //nolint: errcheck
				_, _ = f.Seek(0, io.SeekStart) //nolint: errcheck

				fs.On("Open", mock.Anything).Once().
					Return(f, nil)
			}),
		},
		{
			scenario: "success",
			mockFs: aferomock.MockFs(func(fs *aferomock.Fs) {
				f := mem.NewFileHandle(mem.CreateFile("cookies.json"))
				_, _ = f.WriteString(`{"example.com":{"example.com;;/": {"Name": "id", "Value": "42", "Domain": "example.com", "Path": "/", "SeqNum": 1}}}`) //nolint: errcheck
				_, _ = f.Seek(0, io.SeekStart)                                                                                                               //nolint: errcheck

				fs.On("Open", filePath).Once().
					Return(f, nil)
			}),
			expected: []*http.Cookie{{
				Name:  "id",
				Value: "42",
			}},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.scenario, func(t *testing.T) {
			t.Parallel()

			j := cookiejar.NewPersistentJar(
				cookiejar.WithFs(tc.mockFs(t)),
				cookiejar.WithFilePath(filePath),
				cookiejar.WithLogger(ctxd.NoOpLogger{}),
			)

			u := &url.URL{Scheme: "https", Host: "example.com"}

			actual := j.Cookies(u)

			assert.Equal(t, tc.expected, actual)
		})
	}
}

func TestPersistentJar_Sync(t *testing.T) {
	t.Parallel()

	const filePath = "/tmp/cookies.json"

	testCases := []struct {
		scenario      string
		mockFs        aferomock.FsMocker
		expectedError string
	}{
		{
			scenario: "could not open file for writing",
			mockFs: aferomock.MockFs(func(fs *aferomock.Fs) {
				fs.On("OpenFile", mock.Anything, mock.Anything, mock.Anything).Once().
					Return(nil, errors.New("open file error"))
			}),
			expectedError: "could not open file for persisting cookies: open file error",
		},
		{
			scenario: "could not encode cookies",
			mockFs: aferomock.MockFs(func(fs *aferomock.Fs) {
				f := mem.NewFileHandle(mem.CreateFile("test"))
				_ = f.Close() //nolint: errcheck

				fs.On("OpenFile", mock.Anything, mock.Anything, mock.Anything).Once().
					Return(f, nil)
			}),
			expectedError: "could not serialize cookies: File is closed",
		},
		{
			scenario: "could not sync file",
			mockFs: aferomock.MockFs(func(fs *aferomock.Fs) {
				f := &fileWithSyncError{
					File:      mem.NewFileHandle(mem.CreateFile("test")),
					SyncError: errors.New("sync error"),
				}

				fs.On("OpenFile", mock.Anything, mock.Anything, mock.Anything).Once().
					Return(f, nil)
			}),
			expectedError: "could not sync cookies file: sync error",
		},
		{
			scenario: "success",
			mockFs: aferomock.MockFs(func(fs *aferomock.Fs) {
				fs.On("OpenFile", mock.Anything, mock.Anything, mock.Anything).Once().
					Return(mem.NewFileHandle(mem.CreateFile("test")), nil)
			}),
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.scenario, func(t *testing.T) {
			t.Parallel()

			j := cookiejar.NewPersistentJar(
				cookiejar.WithAutoSync(true),
				cookiejar.WithFs(tc.mockFs(t)),
				cookiejar.WithFilePath(filePath),
				cookiejar.WithLogger(ctxd.NoOpLogger{}),
			)

			err := j.Sync()

			if tc.expectedError == "" {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tc.expectedError)
			}
		})
	}
}

func TestWithSerDer(t *testing.T) {
	t.Parallel()

	fs := aferomock.MockFs(func(fs *aferomock.Fs) {
		fs.On("Open", mock.Anything).Once().
			Return(mem.NewFileHandle(mem.CreateFile("cookies.json")), nil)
	})(t)

	j := cookiejar.NewPersistentJar(
		cookiejar.WithFs(fs),
		cookiejar.WithSerDer(&serder{
			serialize: func(io.Writer, map[string]map[string]cookiejar.Entry) error {
				return nil
			},
			deserialize: func(io.Reader) (map[string]map[string]cookiejar.Entry, error) {
				return map[string]map[string]cookiejar.Entry{
					"example.com": {
						"example.com;;/": {
							Name:   "id",
							Value:  "42",
							Domain: "example.com",
							Path:   "/",
						},
					},
				}, nil
			},
		}),
	)

	u := &url.URL{Scheme: "https", Host: "example.com"}

	actual := j.Cookies(u)
	expected := []*http.Cookie{{
		Name:  "id",
		Value: "42",
	}}

	assert.Equal(t, expected, actual)
}

func readFileData(data *mem.FileData) []byte {
	f := mem.NewFileHandle(data)
	defer f.Close() //nolint: errcheck

	b, err := io.ReadAll(f)
	if err != nil {
		panic(err)
	}

	return b
}

func assertFileDataEqual(t *testing.T, expected string, actual *mem.FileData) {
	t.Helper()

	assert.Equal(t, expected, string(readFileData(actual)))
}

func assertFileDataJSONEqual(t *testing.T, expected string, actual *mem.FileData) {
	t.Helper()

	assertjson.Equal(t, []byte(expected), readFileData(actual))
}

type fileWithSyncError struct {
	afero.File
	SyncError error
}

func (f *fileWithSyncError) Sync() error {
	return f.SyncError
}

type serder struct {
	serialize   func(w io.Writer, entries map[string]map[string]cookiejar.Entry) error
	deserialize func(r io.Reader) (map[string]map[string]cookiejar.Entry, error)
}

func (s *serder) Serialize(w io.Writer, entries map[string]map[string]cookiejar.Entry) error {
	return s.serialize(w, entries)
}

func (s *serder) Deserialize(r io.Reader) (map[string]map[string]cookiejar.Entry, error) {
	return s.deserialize(r)
}
