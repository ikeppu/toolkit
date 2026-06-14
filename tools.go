// Package toolkit provides utility functions for the application.
package toolkit

// Import the cryptographically secure random number generator package.
import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

// Define the alphabet/source characters from which the random string will be generated.
// It contains 64 characters: a-z, A-Z, 0-9, and the characters '_' and '+'.
const randomStringSource = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_+"

// Think about struct as about class, small classes.
// Tools is an empty struct used as a receiver to group utility methods.
type Tools struct {
	MaxFilesSize       int
	AllowedFileTypes   []string
	MaxJsonSize        int
	AllowUnknownFields bool
}

// RandomString generates a cryptographically secure random string of length n.
func (t *Tools) RandomString(n int) string {
	// Initialize a slice of runes 's' of size 'n' to hold the generated string characters.
	// Convert the 'randomStringSource' string to a slice of runes 'r' for efficient indexing.
	s, r := make([]rune, n), []rune(randomStringSource)

	// Loop 'n' times to generate each character of the random string.
	for i := range s {
		// Generate a cryptographically secure random prime number 'p' of bit-length equal to len(r) (64 bits).
		// rand.Reader is the global, shared CSPRNG source.
		p, _ := rand.Prime(rand.Reader, len(r))

		// Convert the random prime number 'p' (big.Int) to a uint64 variable 'x'.
		// Convert the length of the rune slice 'r' (64) to a uint64 variable 'y'.
		x, y := p.Uint64(), uint64(len(r))

		// Use the modulo operator (x % y) to get a random index in the range [0, 63].
		// Assign the character from 'r' at that index to the current position in the slice 's'.
		s[i] = r[x%y]
	}

	// Convert the final slice of runes 's' back into a string and return it.
	return string(s)
}

// UploadedFile is struct used to save information about an uploaded file
type UploadFile struct {
	NewFileName      string
	OriginalFileName string
	FileSize         int64
}

func (t *Tools) UploadFiles(r *http.Request, uploadDir string, rename ...bool) ([]*UploadFile, error) {
	renameFile := true
	if len(rename) > 0 {
		renameFile = rename[0]
	}

	var uploadedFiles []*UploadFile

	if t.MaxFilesSize == 0 {
		t.MaxFilesSize = 1024 * 1024 * 1024
	}

	err := t.CreateDirIfNotExist(uploadDir)
	if err != nil {
		return nil, err
	}

	err = r.ParseMultipartForm(int64(t.MaxFilesSize))

	if err != nil {
		return nil, errors.New("the uploaded file is to big")
	}

	for _, fHeaders := range r.MultipartForm.File {
		for _, hdr := range fHeaders {
			uploadedFiles, err = func(uploadedFiles []*UploadFile) ([]*UploadFile, error) {
				var uploadedFile UploadFile
				infile, err := hdr.Open()
				if err != nil {
					return nil, err
				}
				defer infile.Close()

				buff := make([]byte, 512)
				_, err = infile.Read(buff)

				if err != nil {
					return nil, err
				}

				// check to see if the file type is permitted
				allowed := false
				fileType := http.DetectContentType(buff)
				// allowedTypes := []string{"image/jpeg", "image/png", "image/gif"}

				if len(t.AllowedFileTypes) > 0 {
					for _, x := range t.AllowedFileTypes {
						if strings.EqualFold(fileType, x) {

							allowed = true
						}
					}
				} else {
					allowed = true
				}

				if !allowed {
					return nil, errors.New("the uploaded file type is not permitted")
				}

				_, err = infile.Seek(0, 0)
				if err != nil {
					return nil, err
				}

				if renameFile {
					uploadedFile.NewFileName = fmt.Sprintf("%s%s", t.RandomString(25), filepath.Ext(hdr.Filename))
				} else {
					uploadedFile.NewFileName = hdr.Filename
				}

				var outfile *os.File
				defer outfile.Close()

				if outfile, err = os.Create(filepath.Join(uploadDir, uploadedFile.NewFileName)); err != nil {
					return nil, err
				} else {
					fileSize, err := io.Copy(outfile, infile)
					if err != nil {
						return nil, err
					}

					uploadedFile.FileSize = fileSize
				}

				uploadedFiles = append(uploadedFiles, &uploadedFile)

				return uploadedFiles, nil
			}(uploadedFiles)
			if err != nil {
				return uploadedFiles, err
			}
		}
	}

	return uploadedFiles, nil
}

func (t *Tools) UploadFile(r *http.Request, uploadDir string, rename ...bool) (*UploadFile, error) {

	renameFile := true

	if len(rename) > 0 {
		renameFile = rename[0]
	}

	files, err := t.UploadFiles(r, uploadDir, renameFile)

	if err != nil {
		return nil, err
	}

	return files[0], nil
}

func (t *Tools) CreateDirIfNotExist(path string) error {
	const mode = 0755

	if _, err := os.Stat(path); os.IsNotExist(err) {
		err := os.MkdirAll(path, mode)
		if err != nil {
			return err
		}
	}

	return nil
}

func (t *Tools) Slugify(s string) (string, error) {
	if s == "" {
		return "", errors.New("empty string is not allowed")
	}

	var re = regexp.MustCompile(`[^a-z\d]+`)

	slug := strings.Trim(re.ReplaceAllString(strings.ToLower(s), "-"), "-")

	if len(s) == 0 {
		return "", errors.New("after removing characters, slug is zero length")
	}

	return slug, nil
}

func (t *Tools) DownloadStaticFile(w http.ResponseWriter, r *http.Request, p, file, displayName string) {
	fp := path.Join(p, file)

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", displayName))

	http.ServeFile(w, r, fp)
}

type JSONResponse struct {
	Error   bool        `json:"error"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Read JSON

func (t *Tools) ReadJSON(w http.ResponseWriter, r *http.Request, payload interface{}) error {
	maxBytes := 1024 * 1024 * 1024 // one meg
	if t.MaxJsonSize != 0 {
		maxBytes = t.MaxJsonSize
	}

	r.Body = http.MaxBytesReader(w, r.Body, int64(maxBytes))

	dec := json.NewDecoder(r.Body)

	if !t.AllowUnknownFields {
		dec.DisallowUnknownFields()
	}

	err := dec.Decode(payload)
	if err != nil {
		var syntaxError *json.SyntaxError
		var unmarshalTypeError *json.UnmarshalTypeError
		var unmarshalError *json.InvalidUnmarshalError

		switch {
		case errors.As(err, &syntaxError):
			return fmt.Errorf("body contains badly-formed JSON (at character %d)", syntaxError.Offset)
		case errors.Is(err, io.ErrUnexpectedEOF):
			return fmt.Errorf("body contains badly formed JSON")
		case errors.As(err, &unmarshalTypeError):
			if unmarshalTypeError.Field != "" {
				return fmt.Errorf("body contains incorrect JSON type for field %q", unmarshalTypeError.Field)
			}
			return fmt.Errorf("body contains incorrect JSON type (at character %d)", unmarshalTypeError.Offset)
		case errors.Is(err, io.EOF):
			return fmt.Errorf("body must not be empty")
		case strings.HasPrefix(err.Error(), "json: unknown field"):
			fieldName := strings.TrimPrefix(err.Error(), "json: unknown field")
			return fmt.Errorf("body contains unknown key %s", fieldName)
		case err.Error() == "http: request body to large":
			return fmt.Errorf("body must not be larger than %d bytes", maxBytes)
		case errors.As(err, &unmarshalError):
			return fmt.Errorf("error unmarshalling JSON:%s", err.Error())
		default:
			return err
		}
	}

	err = dec.Decode(&struct{}{})
	if err != io.EOF {
		return errors.New("body must containt only one JSON value")
	}

	return nil
}

// Write Json

func (t *Tools) WriteJSON(w http.ResponseWriter, status int, data interface{}, headers ...http.Header) error {
	out, err := json.Marshal(data)
	if err != nil {
		return err
	}

	if len(headers) > 0 {
		for key, value := range headers[0] {
			w.Header()[key] = value
		}
	}

	w.Header().Set("Content-Type", "Application/json")
	w.WriteHeader(status)

	_, err = w.Write(out)
	if err != nil {
		return err
	}

	return nil
}

func (t *Tools) ErrorJSON(w http.ResponseWriter, err error, status ...int) error {
	statusCode := http.StatusBadRequest

	if len(status) > 0 {
		statusCode = status[0]
	}

	var payload JSONResponse
	payload.Error = true
	payload.Message = err.Error()

	return t.WriteJSON(w, statusCode, payload)
}

func (t *Tools) PushJSONToRemote(uri string, data interface{}, client ...*http.Client) (*http.Response, int, error) {
	// create a json
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, 0, err
	}

	// check for custom http client
	httpClient := &http.Client{}
	if len(client) > 0 {
		httpClient = client[0]
	}

	// build a request and set a header
	request, err := http.NewRequest("POST", uri, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, 0, err
	}
	request.Header.Set("Content-Type", "application/json")

	// call the remote
	response, err := httpClient.Do(request)
	if err != nil {
		return nil, 0, err
	}
	defer response.Body.Close()

	// send the response back
	return response, response.StatusCode, nil
}
