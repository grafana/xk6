package ctrf

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Report struct {
	ReportFormat string    `json:"reportFormat"`
	SpecVersion  string    `json:"specVersion"`
	ReportId     string    `json:"reportId,omitempty"` //nolint:revive // this field is part of the interface: careful.
	Timestamp    time.Time `json:"timestamp,omitempty"`
	GeneratedBy  string    `json:"generatedBy,omitempty"`
	Results      *Results  `json:"results"`
	Extra        any       `json:"extra,omitempty"`
}

const (
	ReportFormatCTRF   = "CTRF"
	SpecVersionCTRF    = "0.0.0"
	GeneratedByDefault = "go-ctrf-json-reporter"
)

func NewReport(toolName string, env *Environment) *Report {
	return &Report{
		ReportFormat: ReportFormatCTRF,
		SpecVersion:  SpecVersionCTRF,
		ReportId:     uuid.New().String(),
		Timestamp:    time.Now(),
		Results: &Results{
			Tool:        &Tool{Name: toolName},
			Environment: env,
			Summary:     &Summary{},
		},
		GeneratedBy: GeneratedByDefault,
	}
}

func (report *Report) ToJson() (string, error) {
	stringBuilder := &strings.Builder{}

	err := report.Write(stringBuilder, false)
	if err != nil {
		return "", err
	}

	return stringBuilder.String(), nil
}

func (report *Report) ToJsonPretty() (string, error) {
	stringBuilder := &strings.Builder{}

	err := report.Write(stringBuilder, true)
	if err != nil {
		return "", err
	}

	return stringBuilder.String(), nil
}

func (report *Report) Write(w io.Writer, pretty bool) error {
	if len(report.Validate()) > 0 {
		return errors.New("report is invalid")
	}
	encoder := json.NewEncoder(w)
	if pretty {
		encoder.SetIndent("", "  ")
	}
	err := encoder.Encode(report)
	if err != nil {
		return fmt.Errorf("error writing ctrf json report: %v", err)
	}
	return nil
}

func (report *Report) WriteFile(filePath string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("error writing ctrf json report: %v", err)
	}
	defer func(file *os.File) {
		_ = file.Close()
	}(file)

	return report.Write(file, true)
}

func (report *Report) Validate() []error {
	results := report.Results
	if results == nil {
		return singleError("missing property 'results'")
	}

	var errs []error
	if results.Tool == nil {
		errs = append(errs, errors.New("missing property 'results.tool'"))
	} else {
		errs = append(errs, results.Tool.Validate()...)
	}
	if results.Summary == nil {
		errs = append(errs, errors.New("missing property 'results.summary'"))
	} else {
		errs = append(errs, results.Summary.Validate()...)
	}
	if results.Tests == nil {
		errs = append(errs, errors.New("missing property 'results.tests'"))
	}
	return errs
}

func singleError(errorMessage string) []error {
	return []error{errors.New(errorMessage)}
}

type Results struct {
	Tool        *Tool         `json:"tool"`
	Summary     *Summary      `json:"summary"`
	Tests       []*TestResult `json:"tests"`
	Environment *Environment  `json:"environment,omitempty"`
	Extra       any           `json:"extra,omitempty"`
}

type Tool struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
	Extra   any    `json:"extra,omitempty"`
}

func (tool *Tool) Validate() []error {
	if tool.Name == "" {
		return singleError("missing property 'results.tool.name'")
	}
	return nil
}

type Summary struct {
	Tests   int   `json:"tests"`
	Passed  int   `json:"passed"`
	Failed  int   `json:"failed"`
	Pending int   `json:"pending"`
	Skipped int   `json:"skipped"`
	Other   int   `json:"other"`
	Suites  int   `json:"suites,omitempty"`
	Start   int64 `json:"start"`
	Stop    int64 `json:"stop"`
	Extra   any   `json:"extra,omitempty"`
}

func (summary *Summary) Validate() []error {
	var errs []error
	if summary.Tests < 0 {
		errs = append(errs, errors.New("invalid property 'results.summary.tests'"))
	}
	if summary.Passed < 0 {
		errs = append(errs, errors.New("invalid property 'results.summary.passed'"))
	}
	if summary.Failed < 0 {
		errs = append(errs, errors.New("invalid property 'results.summary.failed'"))
	}
	if summary.Pending < 0 {
		errs = append(errs, errors.New("invalid property 'results.summary.pending'"))
	}
	if summary.Skipped < 0 {
		errs = append(errs, errors.New("invalid property 'results.summary.skipped'"))
	}
	if summary.Other < 0 {
		errs = append(errs, errors.New("invalid property 'results.summary.other'"))
	}
	if summary.Start < 0 {
		errs = append(errs, errors.New("invalid property 'results.summary.start'"))
	}
	if summary.Stop < 0 {
		errs = append(errs, errors.New("invalid property 'results.summary.stop'"))
	}
	if summary.Suites < 0 {
		errs = append(errs, errors.New("invalid property 'results.summary.suites'"))
	}
	if summary.Start > summary.Stop {
		errs = append(errs, errors.New("invalid summary timestamps: start can't be greater than stop"))
	}
	testsSum := summary.Passed + summary.Failed + summary.Pending + summary.Skipped + summary.Other
	if summary.Tests != testsSum {
		errs = append(errs, fmt.Errorf("invalid summary counts: tests (%d) must be the sum of passed, failed, pending, skipped, and other (%d)", summary.Tests, testsSum))

	}
	return errs
}

type TestStatus string

const (
	TestPassed  TestStatus = "passed"
	TestFailed  TestStatus = "failed"
	TestSkipped TestStatus = "skipped"
	TestPending TestStatus = "pending"
	TestOther   TestStatus = "other"
)

type TestResult struct {
	Name       string     `json:"name"`
	Status     TestStatus `json:"status"`
	Duration   int64      `json:"duration"`
	Start      int64      `json:"start,omitempty"`
	Stop       int64      `json:"stop,omitempty"`
	Suite      string     `json:"suite,omitempty"`
	Message    string     `json:"message,omitempty"`
	Trace      string     `json:"trace,omitempty"`
	RawStatus  string     `json:"rawStatus,omitempty"`
	Tags       []string   `json:"tags,omitempty"`
	Type       string     `json:"type,omitempty"`
	Filepath   string     `json:"filePath,omitempty"`
	Retry      int        `json:"retry,omitempty"`
	Flake      bool       `json:"flake,omitempty"`
	Browser    string     `json:"browser,omitempty"`
	Device     string     `json:"device,omitempty"`
	Screenshot string     `json:"screenshot,omitempty"`
	Parameters any        `json:"parameters,omitempty"`
	Steps      []any      `json:"steps,omitempty"`
	Extra      any        `json:"extra,omitempty"`
}

type Environment struct {
	AppName     string `json:"appName,omitempty"`
	AppVersion  string `json:"appVersion,omitempty"`
	OSPlatform  string `json:"osPlatform,omitempty"`
	OSRelease   string `json:"osRelease,omitempty"`
	OSVersion   string `json:"osVersion,omitempty"`
	BuildName   string `json:"buildName,omitempty"`
	BuildNumber string `json:"buildNumber,omitempty"`
	Extra       any    `json:"extra,omitempty"`
}
