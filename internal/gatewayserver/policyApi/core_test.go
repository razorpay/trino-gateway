package policyapi

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_setIntersection(t *testing.T) {
	s_empty := map[string]struct{}{}
	s_nil := map[string]struct{}(nil)
	s1 := map[string]struct{}{
		"a": {},
		"b": {},
	}
	s2 := map[string]struct{}{
		"c": {},
		"d": {},
	}
	s3 := map[string]struct{}{
		"a": {},
		"d": {},
	}
	s13 := map[string]struct{}{
		"a": {},
	}
	s23 := map[string]struct{}{
		"d": {},
	}

	// both empty sets
	assert.Equal(t, s_empty, setIntersection(s_empty, s_empty))

	// first set empty
	assert.Equal(t, s_empty, setIntersection(s_empty, s1))

	// second set empty
	assert.Equal(t, s_empty, setIntersection(s1, s_empty))

	// both non-empty sets with empty intersection
	assert.Equal(t, s_empty, setIntersection(s1, s2))

	// both non-empty sets with non-empty intersection
	assert.Equal(t, s13, setIntersection(s1, s3))
	assert.Equal(t, s23, setIntersection(s3, s2))
	// reverse order
	assert.Equal(t, s13, setIntersection(s3, s1))
	assert.Equal(t, s23, setIntersection(s2, s3))

	// one set nil
	assert.Equal(t, s3, setIntersection(s_nil, s3))

	// both sets nil
	assert.Equal(t, s_nil, setIntersection(s_nil, s_nil))

	// nested stuff
	assert.Equal(t, s13, setIntersection(setIntersection(setIntersection(s1, s_nil), s_nil), s3))
}
