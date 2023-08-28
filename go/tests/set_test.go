package tests_test

import (
	"github.com/rulestone/immutable"
	"github.com/rulestone/types"
	"testing"
)

func TestImmutableSets(t *testing.T) {
	s1 := immutable.Of[types.Category](10, 20, 30)
	s2 := immutable.Of[types.Category](10, 20, 40)
	s3 := immutable.Of[immutable.Set[types.Category]](*s1, *s2)
	s4 := s1.Union(*s2)

	if s1.Has(10) == false {
		t.Fatalf("failed")
	}

	if s2.Has(40) == false {
		t.Fatalf("failed")
	}

	if s3.Has(*s1) == false {
		t.Fatalf("failed")
	}

	if s4.Has(30) == false {
		t.Fatalf("failed")
	}

	if s4.Has(40) == false {
		t.Fatalf("failed")
	}
}
