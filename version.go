package main

import (
	"fmt"
	"regexp"
	"strconv"
	"time"
)

var versionPattern = regexp.MustCompile(`^v(\d+)\.(\d+)-(\d{12})$`)

type Version struct {
	Raw          string
	Major        int
	Minor        int
	RevisionTime time.Time
}

func ParseVersion(raw string) (Version, error) {
	matches := versionPattern.FindStringSubmatch(raw)
	if matches == nil {
		return Version{}, fmt.Errorf("invalid version format: %s", raw)
	}

	major, err := strconv.Atoi(matches[1])
	if err != nil {
		return Version{}, fmt.Errorf("parse major version: %w", err)
	}

	minor, err := strconv.Atoi(matches[2])
	if err != nil {
		return Version{}, fmt.Errorf("parse minor version: %w", err)
	}

	revisionTime, err := time.Parse("200601021504", matches[3])
	if err != nil {
		return Version{}, fmt.Errorf("parse revision time: %w", err)
	}

	return Version{
		Raw:          raw,
		Major:        major,
		Minor:        minor,
		RevisionTime: revisionTime,
	}, nil
}

func CompareVersion(a, b Version) int {
	switch {
	case a.Major < b.Major:
		return -1
	case a.Major > b.Major:
		return 1
	case a.Minor < b.Minor:
		return -1
	case a.Minor > b.Minor:
		return 1
	case a.RevisionTime.Before(b.RevisionTime):
		return -1
	case a.RevisionTime.After(b.RevisionTime):
		return 1
	default:
		return 0
	}
}
