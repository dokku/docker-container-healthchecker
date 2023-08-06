package convert

import (
	"docker-container-healthchecker/appjson"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type ChecksFile struct {
	data     []byte
	checks   []Check
	wait     int
	timeout  int
	attempts int
}

type Check struct {
	Content string
	Host    string
	Path    string
	Scheme  string
}

func New(options ...func(*ChecksFile)) *ChecksFile {
	svr := &ChecksFile{}
	for _, o := range options {
		o(svr)
	}
	return svr
}

func (s *ChecksFile) ToHealthchecks() []appjson.Healthcheck {
	var healthchecks []appjson.Healthcheck
	for i, check := range s.checks {
		headers := []appjson.HTTPHeader{}
		if len(check.Host) > 0 {
			headers = append(headers, appjson.HTTPHeader{
				Name:  "Host",
				Value: check.Host,
			})
		}

		scheme := ""
		if check.Scheme != "http" {
			scheme = check.Scheme
		}

		healthchecks = append(healthchecks, appjson.Healthcheck{
			Attempts:    s.attempts,
			Content:     check.Content,
			HTTPHeaders: headers,
			Name:        fmt.Sprintf("check-%d", i+1),
			Path:        check.Path,
			Scheme:      scheme,
			Timeout:     s.timeout,
			Type:        "startup",
			Wait:        s.wait,
		})
	}

	return healthchecks
}

func (s *ChecksFile) Parse() error {
	r, err := regexp.Compile("^[^#]*")
	if err != nil {
		return fmt.Errorf("unable to parse line regex: %w", err)
	}

	var re = regexp.MustCompile(`(?mi)^(?:https?:)?(\/\/[^\/\?]+)`)

	for i, line := range strings.Split(string(s.data), "\n") {
		line := r.FindString(line)
		line = strings.TrimSpace(line)
		if len(line) == 0 || strings.HasPrefix(line, "#") {
			continue
		}

		if strings.HasPrefix(line, "ATTEMPTS=") {
			value, err := parseEnvVarToInt(line, i)
			if err != nil {
				return err
			}

			s.attempts = value
		} else if strings.HasPrefix(line, "TIMEOUT=") {
			value, err := parseEnvVarToInt(line, i)
			if err != nil {
				return err
			}

			s.timeout = value
		} else if strings.HasPrefix(line, "WAIT=") {
			value, err := parseEnvVarToInt(line, i)
			if err != nil {
				return err
			}

			s.wait = value
		} else {
			parts := strings.SplitN(line, " ", 2)
			path := parts[0]

			scheme := "http"
			if strings.HasPrefix(path, "https://") {
				scheme = "https"
				path = strings.TrimPrefix(path, "https:")
			} else if strings.HasPrefix(path, "http://") {
				scheme = "http"
				path = strings.TrimPrefix(path, "http:")
			}

			hostname := ""
			if strings.HasPrefix(path, "//") {
				hostname = strings.TrimPrefix(re.FindString(path), "//")
				path = string(re.ReplaceAll([]byte(path), []byte("")))

			}

			if len(parts) == 1 {
				s.checks = append(s.checks, Check{
					Host:   hostname,
					Path:   path,
					Scheme: scheme,
				})
			} else {
				s.checks = append(s.checks, Check{
					Host:    hostname,
					Content: parts[1],
					Path:    path,
					Scheme:  scheme,
				})
			}
		}
	}

	return nil
}

func WithData(data []byte) func(*ChecksFile) {
	return func(s *ChecksFile) {
		s.data = data
	}
}

func parseEnvVarToInt(line string, line_number int) (int, error) {
	parts := strings.SplitN(line, "=", 2)
	if len(parts) != 2 {
		return 0, fmt.Errorf("error parsing line %d: %s", line_number, line)
	}

	value, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, fmt.Errorf("error parsing WAIT value at line %d: %w", line_number, err)
	}

	return value, nil
}
