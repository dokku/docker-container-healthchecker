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
	Path    string
	Content string
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
	for _, check := range s.checks {
		healthchecks = append(healthchecks, appjson.Healthcheck{
			Attempts: s.attempts,
			Path:     check.Path,
			Content:  check.Content,
			Timeout:  s.timeout,
			Wait:     s.wait,
		})
	}

	return healthchecks
}

func (s *ChecksFile) Parse() error {
	r, err := regexp.Compile("^[^#]*")
	if err != nil {
		return fmt.Errorf("unable to parse line regex: %w", err)
	}

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
			if len(parts) == 1 {
				s.checks = append(s.checks, Check{
					Path: parts[0],
				})
			} else {
				s.checks = append(s.checks, Check{
					Path:    parts[0],
					Content: parts[1],
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
