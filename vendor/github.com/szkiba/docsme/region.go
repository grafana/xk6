package docsme

// code reused from https://github.com/szkiba/mdcode/blob/master/internal/region/region.go

import (
	"fmt"
	"regexp"
)

const (
	reSpec       = `[!"#$%%&'()*+,\-./:;<=>?@[\\\]^_{|}~]`
	reLineBegin  = `(?m)^[[:blank:]]*`
	reLineEnd    = `*[[:blank:]]*\r?\n`
	regionFormat = reLineBegin + reSpec +
		`+[[:blank:]]*#region[[:blank:]]+%s[[:blank:]]*` +
		reSpec + reLineEnd
	namedendFormat = reLineBegin + reSpec +
		`+[[:blank:]]*#endregion[[:blank:]]+%s[[:blank:]]*` +
		reSpec + reLineEnd
)

var reEnd = regexp.MustCompile(reLineBegin + reSpec +
	`+[[:blank:]]*#endregion[[:blank:]]*` +
	reSpec + reLineEnd)

func marker(format string, name string) (*regexp.Regexp, error) {
	return regexp.Compile(fmt.Sprintf(format, regexp.QuoteMeta(name)))
}

func findRegion(source []byte, name string) (bool, int, int, error) {
	reBegin, err := marker(regionFormat, name)
	if err != nil {
		return false, 0, 0, err
	}

	idxBegin := reBegin.FindIndex(source)
	if idxBegin == nil {
		return false, 0, 0, nil
	}

	namedEnd, err := marker(namedendFormat, name)
	if err != nil {
		return false, 0, 0, err
	}

	idxEnd := namedEnd.FindIndex(source[idxBegin[1]:])
	if idxEnd == nil {
		idxEnd = reEnd.FindIndex(source[idxBegin[1]:])
		if idxEnd == nil {
			return false, 0, 0, nil
		}
	}

	return true, idxBegin[1], idxBegin[1] + idxEnd[0], nil
}

func replace(source []byte, name string, value []byte) ([]byte, bool, error) {
	found, begin, end, err := findRegion(source, name)
	if err != nil {
		return nil, false, err
	}

	if !found {
		return nil, false, nil
	}

	res := make([]byte, len(source)-(end-begin)+len(value))

	copy(res, source[:begin])
	copy(res[begin:], value)
	copy(res[begin+len(value):], source[end:])

	return res, true, nil
}
