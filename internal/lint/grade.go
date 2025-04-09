package lint

import (
	"fmt"
	"os"
	"strings"
)

//nolint:gochecknoglobals,mnd
var gradeMins = map[Grade]int{
	GradeA: 95,
	GradeB: 90,
	GradeC: 75,
	GradeD: 60,
	GradeE: 45,
	GradeF: 30,
	GradeG: 1,
	GradeZ: 0,
}

func gradeFor(level int) Grade {
	for _, grade := range []Grade{GradeA, GradeB, GradeC, GradeD, GradeE, GradeF, GradeG} {
		if level >= gradeMins[grade] {
			return grade
		}
	}

	return GradeZ
}

func (g *Grade) String() string {
	return string(*g)
}

// Set implements cobra.Value#Set().
func (g *Grade) Set(val string) error {
	v := strings.ToUpper(val)

	switch v {
	case "A", "B", "C", "D", "E", "F", "G", "Z":
		*g = Grade(v)
	default:
		return fmt.Errorf("%w: %s", os.ErrInvalid, val)
	}

	return nil
}

// Type implements cobra.Value#Type().
func (g *Grade) Type() string {
	return "A|B|C|D|E|F|G|Z"
}
