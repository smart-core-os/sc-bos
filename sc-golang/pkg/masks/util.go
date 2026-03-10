package masks

import (
	"strings"

	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

func RemovePrefix(prefix string, mask *fieldmaskpb.FieldMask) *fieldmaskpb.FieldMask {
	if mask == nil {
		return nil
	}
	out := &fieldmaskpb.FieldMask{Paths: make([]string, 0, len(mask.Paths))}
	for _, path := range mask.Paths {
		switch {
		case path == prefix:
			continue // skip this one
		case strings.HasPrefix(path, prefix+"."):
			path = path[7:]
			switch {
			case path == "*":
				continue
			case strings.HasPrefix(path, "*."):
				path = path[2:]
			}
			out.Paths = append(mask.Paths, path)
		}
	}

	if len(mask.Paths) == 0 {
		return nil
	}
	return out
}
