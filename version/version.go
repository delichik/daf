package version

import (
	"strconv"
	"strings"
)

type Version struct {
	Major int
	Minor int
	Build int
}

func Parse(version string) *Version {
	parts := strings.Split(version, ".")
	if len(parts) != 3 {
		return &Version{}
	}
	var err error
	v := &Version{}
	v.Major, _ = strconv.Atoi(parts[0])
	if err != nil {
		return &Version{}
	}
	v.Minor, _ = strconv.Atoi(parts[1])
	if err != nil {
		return &Version{}
	}
	v.Build, _ = strconv.Atoi(parts[2])
	if err != nil {
		return &Version{}
	}
	return v
}

func (v *Version) IsNewThan(v2 *Version) bool {
	if v.Major > v2.Major {
		return true
	}
	if v.Major < v2.Major {
		return false
	}
	if v.Minor > v2.Minor {
		return true
	}
	if v.Minor < v2.Minor {
		return false
	}
	if v.Build > v2.Build {
		return true
	}
	if v.Build < v2.Build {
		return false
	}
	return false
}
