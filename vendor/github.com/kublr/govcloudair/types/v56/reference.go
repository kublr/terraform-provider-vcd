package types

// ReferencePredicate is a predicate for finding reference in a reference list
type ReferencePredicate func(*Reference) bool

// ReferenceList represents a list of references
type ReferenceList []*Reference

// Find the first occurrence that matches the predicate
func (l ReferenceList) Find(predicate ReferencePredicate) *Reference {
	for _, ref := range l {
		if predicate(ref) {
			return ref
		}
	}

	return nil
}

// ForName finds a reference for a given name
func (l ReferenceList) ForName(refName string) *Reference {
	return l.Find(func(ref *Reference) bool {
		return ref.Name == refName
	})
}
