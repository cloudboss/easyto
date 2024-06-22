package aws

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/cloudboss/easyto/pkg/initial/vmspec"
	yaml "github.com/goccy/go-yaml"
)

const (
	errorCodeBodyUnreadable = iota
	errorCodeInvalidMethod
	errorCodeInvalidURL
	errorCodeRequestError
	errorCodeStatusError
)

const (
	endpointMetadataDefault = "169.254.169.254"
)

type HTTPError struct {
	errorCode  int
	statusCode int
	url        string
	wrapped    error
}

func (h *HTTPError) Error() string {
	switch h.errorCode {
	case errorCodeInvalidMethod:
		return "invalid HTTP method"
	case errorCodeInvalidURL:
		return fmt.Errorf("invalid URL %s", h.url).Error()
	case errorCodeBodyUnreadable:
		return fmt.Errorf("unable to read response body: %w",
			h.wrapped).Error()
	case errorCodeRequestError:
		return fmt.Errorf("request error: %w", h.wrapped).Error()
	case errorCodeStatusError:
		return fmt.Errorf("request failed with status %s",
			http.StatusText(h.statusCode)).Error()
	default:
		return "unknown error making http request"
	}
}

func request(method string, requestURL string, header http.Header) (*http.Response, error) {
	u, err := url.Parse(requestURL)
	if err != nil {
		return nil, &HTTPError{errorCode: errorCodeInvalidURL, url: requestURL}
	}

	req := &http.Request{
		URL:    u,
		Header: header,
	}

	switch method {
	case "GET", "PUT":
		req.Method = method
	default:
		return nil, &HTTPError{errorCode: errorCodeInvalidMethod}
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, &HTTPError{errorCode: errorCodeRequestError, wrapped: err}
	}

	if isErrorStatus(resp.StatusCode) {
		resp.Body.Close()
		return nil, &HTTPError{errorCode: errorCodeStatusError, statusCode: resp.StatusCode}
	}
	return resp, nil
}

func requestString(method string, requestURL string, header http.Header) (string, error) {
	resp, err := request(method, requestURL, header)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", &HTTPError{errorCode: errorCodeBodyUnreadable, wrapped: err}
	}
	return string(body), nil
}

func getIMDSv2(path string, endpoint string) (*http.Response, error) {
	tokenURL := &url.URL{Scheme: "http",
		Host: endpoint,
		Path: "/latest/api/token",
	}
	token, err := requestString("PUT", tokenURL.String(), http.Header{
		"X-aws-ec2-metadata-token-ttl-seconds": []string{"21600"},
	})
	if err != nil {
		return nil, fmt.Errorf("unable to get IMDSv2 token: %w", err)
	}

	pathURL := &url.URL{Scheme: "http",
		Host: endpoint,
		Path: path,
	}
	return request("GET", pathURL.String(), http.Header{
		"X-aws-ec2-metadata-token": []string{token},
	})
}

func getIMDSv2String(path string, endpoint string) (string, error) {
	resp, err := getIMDSv2(path, endpoint)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", &HTTPError{errorCode: errorCodeBodyUnreadable, wrapped: err}
	}
	return string(body), nil
}

func GetUserData(endpoint ...string) (*vmspec.VMSpec, error) {
	endpoint0 := endpointMetadataDefault
	if len(endpoint) > 0 {
		endpoint0 = endpoint[0]
	}

	spec := &vmspec.VMSpec{}

	resp, err := getIMDSv2("/latest/user-data", endpoint0)
	if err != nil {
		// Return an empty VMSpec when no user data is defined.
		hErr := &HTTPError{}
		if errors.As(err, &hErr) && hErr.statusCode == http.StatusNotFound {
			slog.Error("Got http error", "error", hErr)
			return spec, nil
		} else {
			slog.Error("Got error", "error", err)
			return nil, err
		}
	}

	err = yaml.NewDecoder(resp.Body).Decode(spec)
	if err != nil {
		return nil, err
	}

	return spec, nil
}

func GetSSHPubKey(endpoint ...string) (string, error) {
	endpoint0 := endpointMetadataDefault
	if len(endpoint) > 0 {
		endpoint0 = endpoint[0]
	}

	return getIMDSv2String("/latest/meta-data/public-keys/0/openssh-key", endpoint0)
}

func GetRegion(endpoint ...string) (string, error) {
	endpoint0 := endpointMetadataDefault
	if len(endpoint) > 0 {
		endpoint0 = endpoint[0]
	}

	return getIMDSv2String("/latest/meta-data/placement/region", endpoint0)
}

func isErrorStatus(status int) bool {
	return status >= http.StatusBadRequest
}
