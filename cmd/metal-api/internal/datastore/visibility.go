package datastore

import (
	"fmt"
	"strings"
)

const (
	// VisibilityPrivate includes resources that are bound to a specific tenants and projects.
	// (e.g. private machine networks)
	VisibilityPrivate Predicate = "private"
	// VisibilityShared includes resources that are bound to a specific tenants and projects when
	// they have set a special "shared" flag. These entities can be read and used by other
	// tenants in other projects. (e.g. private shared networks)
	VisibilityShared Predicate = "shared"
	// VisibilityPublic includes resources provided by administrators that can be read and used
	// by arbitrary tenants in arbitrary projects. (e.g. public machine images)
	VisibilityPublic Predicate = "public"
	// VisibilityAdmin includes resources that can only be read and used by administrators.
	// (e.g. IPMI console data, machine firmware management, ...)
	VisibilityAdmin Predicate = "admin"
)

func visibilityFromName(input string) (Predicates, error) {
	if input == "" {
		return Predicates{VisibilityPrivate, VisibilityShared}, nil
	}

	names := strings.Split(input, ",")

	ps := map[Predicate]bool{}
	for _, name := range names {
		switch name {
		case VisibilityPrivate.String():
			ps[VisibilityPrivate] = true
		case VisibilityShared.String():
			ps[VisibilityShared] = true
		case VisibilityPublic.String():
			ps[VisibilityPublic] = true
		case VisibilityAdmin.String():
			ps[VisibilityAdmin] = true
		default:
			return nil, fmt.Errorf("unsupported visibility: %s", name)
		}
	}

	var result Predicates
	for p := range ps {
		result = append(result, p)
	}

	return result, nil
}
