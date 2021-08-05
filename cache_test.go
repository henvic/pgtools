package pgtools

import (
	"container/list"
	"reflect"
	"testing"
)

func TestWildcardCache(t *testing.T) {
	old := wildcardsCache
	t.Cleanup(func() {
		wildcardsCache = old // Restore default caching.
	})

	const maxCached = 3
	wildcardsCache = &lru{
		cap: maxCached,

		m: map[reflect.Type]*list.Element{},
		l: list.New(),
	}

	mocks := []struct {
		v           interface{}
		want        string
		cachedItems int
	}{
		{
			v: struct {
				Automatic string
				Tagged    string `db:"tagged"`
				OneTwo    string // OneTwo should be one_two in the database.
				CamelCase string `db:"CamelCase"` // CamelCase should not be normalized to camel_case.
				Ignored   string `db:"-"`
			}{},
			want:        `"automatic","tagged","one_two","CamelCase"`,
			cachedItems: 1,
		},
		{
			v: struct {
				Automatic string
				Tagged    string `db:"tagged"`
				OneTwo    string // OneTwo should be one_two in the database.
				CamelCase string `db:"CamelCase"` // CamelCase should not be normalized to camel_case.
				Ignored   string `db:"-"`
			}{},
			want:        `"automatic","tagged","one_two","CamelCase"`,
			cachedItems: 1,
		},
		{
			v: struct {
				Number int
			}{},
			want:        `"number"`,
			cachedItems: 2,
		},
		{
			v: struct {
				Name string
			}{},
			want:        `"name"`,
			cachedItems: 3,
		},
		{
			v: struct {
				A string
				B string
				C string
			}{},
			want:        `"a","b","c"`,
			cachedItems: 3,
		},
		{
			v: struct {
				Name string
			}{},
			want:        `"name"`,
			cachedItems: 3,
		},
		{
			v: struct {
				Name string
				Age  int
			}{},
			want:        `"name","age"`,
			cachedItems: 3,
		},
	}
	for _, m := range mocks {
		orig := Wildcard(m.v)
		cached := Wildcard(m.v)

		if orig != m.want {
			t.Errorf("wanted %v, got %v instead", m.want, orig)
		}
		if orig != cached {
			t.Errorf("wanted cached value %v, got %v instead", m.want, cached)
		}
		if wildcardsCache.l.Len() != len(wildcardsCache.m) {
			t.Error("cache doubly linked list and map length should match")
		}
		if len(wildcardsCache.m) > maxCached {
			t.Errorf("cache should contain %d once full, got %d instead", maxCached, len(wildcardsCache.m))
		}
		if len(wildcardsCache.m) != m.cachedItems {
			t.Errorf("wanted %d cached items, found %d", m.cachedItems, len(wildcardsCache.m))
		}
	}
}
