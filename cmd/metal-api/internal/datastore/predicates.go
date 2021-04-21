package datastore

type Predicate string

func (p Predicate) String() string {
	return string(p)
}

type Predicates []Predicate

// Contains returns true when a given predicate is contained in the slice of predicates.
func (ps Predicates) Contains(p Predicate) bool {
	for _, pp := range ps {
		if pp == p {
			return true
		}
	}
	return false
}

func (p Predicate) Is(pp Predicate) bool {
	return string(p) == string(pp)
}
