package curllogic

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	//"os"
	"strings"
)

// curlargs represents parse command-line arguments for curl
type CurlArgs struct {
	URL string
	Method string
	Headers map[string]string
	Data []byte // for POST or other methods with body
}

// performs the HTTP request based on CurlArgs
func ExecuteCurl(args CurlArgs, output io.Writer) error {
	// Default to GET if no method specified
	if args.Method == "" {
		args.Method = "GET"
	}

	// create HTTP request
	req, err := http.NewRequest(args.Method, args.URL, bytes.NewReader(args.Data))
	if err != nil {
		return fmt.Errorf("Failed to create request: %v", err)
	}

	// add headers
	for key, value := range args.Headers {
		req.Header.Set(key, value)
	}

	// execute request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	// write response to stdout
	_, err = io.Copy(output, resp.Body) // Changed to use output instead of os.Stdout
	if err != nil {
		return fmt.Errorf("Failed to read response: %v", err)
	}

	// append a newline after the response
	_, err = output.Write([]byte("\n"))
	if err != nil {
		return fmt.Errorf("Failed to write newline: %v", err)
	}

	return nil
}

var ErrNoArg = errors.New("argument required")

// parses the command-line input into CurlArgs
func ParseCurlArgs(args []string, pipedInput []byte) (CurlArgs, error) {
    result := CurlArgs{
        Headers: make(map[string]string),
    }

    if len(args) < 2 {
        return result, ErrNoArg
    }

    i := 1 // skip "curl"
    for i < len(args) {
        switch args[i] {
        case "-X":
            if i + 1 >= len(args) {
                return result, fmt.Errorf("-X requires a method")
            }
            result.Method = strings.ToUpper(args[i + 1])
            i += 2
        case "-H":
            if i + 1 >= len(args) {
                return result, fmt.Errorf("-H requires a header")
            }
            headerParts := strings.SplitN(args[i + 1], ":", 2)
            if len(headerParts) != 2 {
                return result, fmt.Errorf("invalid header format: %s", args[i + 1])
            }
			// encapsulating strings.TrimSpace(headerParts[]) within strings.Trim
			// and removing the `"` fixes the issue of not identifying the args
			// correctly. why? idk, prob cause the `"`` were kept in the args
            key := strings.Trim(strings.TrimSpace(headerParts[0]), `"`)
            value := strings.Trim(strings.TrimSpace(headerParts[1]), `"`)
            result.Headers[key] = value
            i += 2
        default:
            // Defer URL assignment until we’ve processed all flags
            i++
        }
    }

    // After processing flags, the last non-flag argument is the URL
    for j := len(args) - 1; j > 0; j-- {
        if args[j] != "-X" && args[j] != "-H" && (j == 0 || args[j - 1] != "-X" && args[j - 1] != "-H") {
            result.URL = args[j]
            break
        }
    }

    if result.URL == "" {
        return result, fmt.Errorf("URL required")
    }

    if len(pipedInput) > 0 && (result.Method == "POST" || result.Method == "PUT") {
        result.Data = pipedInput
        if _, exists := result.Headers["Content-Type"]; !exists {
            result.Headers["Content-Type"] = "application/octet-stream"
        }
    }

    return result, nil
}