package middlewares

import (
	"strings"
)

var paths []Path

type Path struct {
	value    string
	segments []string
	params   map[int]int
}

func MakePath(route string) {
	segments := strings.Split(strings.TrimPrefix(route, "/"), "/")
	path := Path{
		value:    route,
		segments: segments,
		params:   make(map[int]int),
	}

	for i, segment := range segments {
		if strings.HasPrefix(segment, "{") && strings.HasSuffix(segment, "}") {
			path.params[i] = i
		}
	}

	paths = append(paths, path)
}

func GetPath(route string) string {
	segments := strings.Split(strings.TrimPrefix(route, "/"), "/")

	for _, path := range paths {
		if len(path.segments) != len(segments) {
			continue
		}

		matches := true
		for i, segment := range segments {
			if path.segments[i] != segment && !strings.HasPrefix(path.segments[i], "{") {
				matches = false
				break
			}
		}

		if matches {
			return path.value
		}
	}

	return route
}
