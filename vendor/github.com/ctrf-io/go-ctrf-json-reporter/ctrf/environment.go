package ctrf

import (
	"encoding/json"
	"fmt"
	"strconv"
)

// MarshalJSON preserves the string API while emitting CTRF's integer build number.
func (env Environment) MarshalJSON() ([]byte, error) {
	type alias Environment
	output := struct {
		alias

		BuildNumber *int64 `json:"buildNumber,omitempty"`
		Extra       any    `json:"extra,omitempty"`
	}{alias: alias(env), Extra: env.Extra}
	if env.BuildNumber != "" {
		if number, err := strconv.ParseInt(env.BuildNumber, 10, 64); err == nil {
			output.BuildNumber = &number
		} else {
			extra, err := env.metadataWithBuildNumber()
			if err != nil {
				return nil, err
			}
			output.Extra = extra
		}
	}
	data, err := json.Marshal(output)
	if err != nil {
		return nil, fmt.Errorf("encode environment: %w", err)
	}
	return data, nil
}

// UnmarshalJSON accepts both legacy strings and CTRF integer build numbers.
func (env *Environment) UnmarshalJSON(data []byte) error {
	type alias Environment
	input := struct {
		*alias

		BuildNumber json.RawMessage `json:"buildNumber"`
	}{alias: (*alias)(env)}
	if err := json.Unmarshal(data, &input); err != nil {
		return fmt.Errorf("decode environment: %w", err)
	}
	if len(input.BuildNumber) == 0 {
		return nil
	}
	if string(input.BuildNumber) == "null" {
		env.BuildNumber = ""
		return nil
	}
	if err := json.Unmarshal(input.BuildNumber, &env.BuildNumber); err == nil {
		return nil
	}
	var number int64
	if err := json.Unmarshal(input.BuildNumber, &number); err != nil {
		return fmt.Errorf("decode build number: %w", err)
	}
	env.BuildNumber = strconv.FormatInt(number, 10)
	return nil
}

// metadataWithBuildNumber copies object metadata before adding a legacy build identifier.
func (env Environment) metadataWithBuildNumber() (map[string]json.RawMessage, error) {
	metadata, err := json.Marshal(env.Extra)
	if err != nil {
		return nil, fmt.Errorf("encode environment metadata: %w", err)
	}
	extra := make(map[string]json.RawMessage)
	if err := json.Unmarshal(metadata, &extra); err != nil {
		return nil, fmt.Errorf("encode build identifier in environment metadata: %w", err)
	}
	if extra == nil {
		extra = make(map[string]json.RawMessage)
	}
	buildNumber, err := json.Marshal(env.BuildNumber)
	if err != nil {
		return nil, fmt.Errorf("encode build identifier: %w", err)
	}
	extra["buildNumber"] = buildNumber
	return extra, nil
}
