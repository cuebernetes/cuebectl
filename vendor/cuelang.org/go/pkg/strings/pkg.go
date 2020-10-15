// Code generated by go generate. DO NOT EDIT.

//go:generate rm pkg.go
//go:generate go run ../gen/gen.go

package strings

import (
	"cuelang.org/go/internal/core/adt"
	"cuelang.org/go/pkg/internal"
)

func init() {
	internal.Register("strings", pkg)
}

var _ = adt.TopKind // in case the adt package isn't used

var pkg = &internal.Package{
	Native: []*internal.Builtin{{
		Name:   "ByteAt",
		Params: []adt.Kind{adt.BytesKind | adt.StringKind, adt.IntKind},
		Result: adt.IntKind,
		Func: func(c *internal.CallCtxt) {
			b, i := c.Bytes(0), c.Int(1)
			if c.Do() {
				c.Ret, c.Err = ByteAt(b, i)
			}
		},
	}, {
		Name:   "ByteSlice",
		Params: []adt.Kind{adt.BytesKind | adt.StringKind, adt.IntKind, adt.IntKind},
		Result: adt.BytesKind | adt.StringKind,
		Func: func(c *internal.CallCtxt) {
			b, start, end := c.Bytes(0), c.Int(1), c.Int(2)
			if c.Do() {
				c.Ret, c.Err = ByteSlice(b, start, end)
			}
		},
	}, {
		Name:   "Runes",
		Params: []adt.Kind{adt.StringKind},
		Result: adt.ListKind,
		Func: func(c *internal.CallCtxt) {
			s := c.String(0)
			if c.Do() {
				c.Ret = Runes(s)
			}
		},
	}, {
		Name:   "MinRunes",
		Params: []adt.Kind{adt.StringKind, adt.IntKind},
		Result: adt.BoolKind,
		Func: func(c *internal.CallCtxt) {
			s, min := c.String(0), c.Int(1)
			if c.Do() {
				c.Ret = MinRunes(s, min)
			}
		},
	}, {
		Name:   "MaxRunes",
		Params: []adt.Kind{adt.StringKind, adt.IntKind},
		Result: adt.BoolKind,
		Func: func(c *internal.CallCtxt) {
			s, max := c.String(0), c.Int(1)
			if c.Do() {
				c.Ret = MaxRunes(s, max)
			}
		},
	}, {
		Name:   "ToTitle",
		Params: []adt.Kind{adt.StringKind},
		Result: adt.StringKind,
		Func: func(c *internal.CallCtxt) {
			s := c.String(0)
			if c.Do() {
				c.Ret = ToTitle(s)
			}
		},
	}, {
		Name:   "ToCamel",
		Params: []adt.Kind{adt.StringKind},
		Result: adt.StringKind,
		Func: func(c *internal.CallCtxt) {
			s := c.String(0)
			if c.Do() {
				c.Ret = ToCamel(s)
			}
		},
	}, {
		Name:   "SliceRunes",
		Params: []adt.Kind{adt.StringKind, adt.IntKind, adt.IntKind},
		Result: adt.StringKind,
		Func: func(c *internal.CallCtxt) {
			s, start, end := c.String(0), c.Int(1), c.Int(2)
			if c.Do() {
				c.Ret, c.Err = SliceRunes(s, start, end)
			}
		},
	}, {
		Name:   "Compare",
		Params: []adt.Kind{adt.StringKind, adt.StringKind},
		Result: adt.IntKind,
		Func: func(c *internal.CallCtxt) {
			a, b := c.String(0), c.String(1)
			if c.Do() {
				c.Ret = Compare(a, b)
			}
		},
	}, {
		Name:   "Count",
		Params: []adt.Kind{adt.StringKind, adt.StringKind},
		Result: adt.IntKind,
		Func: func(c *internal.CallCtxt) {
			s, substr := c.String(0), c.String(1)
			if c.Do() {
				c.Ret = Count(s, substr)
			}
		},
	}, {
		Name:   "Contains",
		Params: []adt.Kind{adt.StringKind, adt.StringKind},
		Result: adt.BoolKind,
		Func: func(c *internal.CallCtxt) {
			s, substr := c.String(0), c.String(1)
			if c.Do() {
				c.Ret = Contains(s, substr)
			}
		},
	}, {
		Name:   "ContainsAny",
		Params: []adt.Kind{adt.StringKind, adt.StringKind},
		Result: adt.BoolKind,
		Func: func(c *internal.CallCtxt) {
			s, chars := c.String(0), c.String(1)
			if c.Do() {
				c.Ret = ContainsAny(s, chars)
			}
		},
	}, {
		Name:   "LastIndex",
		Params: []adt.Kind{adt.StringKind, adt.StringKind},
		Result: adt.IntKind,
		Func: func(c *internal.CallCtxt) {
			s, substr := c.String(0), c.String(1)
			if c.Do() {
				c.Ret = LastIndex(s, substr)
			}
		},
	}, {
		Name:   "IndexAny",
		Params: []adt.Kind{adt.StringKind, adt.StringKind},
		Result: adt.IntKind,
		Func: func(c *internal.CallCtxt) {
			s, chars := c.String(0), c.String(1)
			if c.Do() {
				c.Ret = IndexAny(s, chars)
			}
		},
	}, {
		Name:   "LastIndexAny",
		Params: []adt.Kind{adt.StringKind, adt.StringKind},
		Result: adt.IntKind,
		Func: func(c *internal.CallCtxt) {
			s, chars := c.String(0), c.String(1)
			if c.Do() {
				c.Ret = LastIndexAny(s, chars)
			}
		},
	}, {
		Name:   "SplitN",
		Params: []adt.Kind{adt.StringKind, adt.StringKind, adt.IntKind},
		Result: adt.ListKind,
		Func: func(c *internal.CallCtxt) {
			s, sep, n := c.String(0), c.String(1), c.Int(2)
			if c.Do() {
				c.Ret = SplitN(s, sep, n)
			}
		},
	}, {
		Name:   "SplitAfterN",
		Params: []adt.Kind{adt.StringKind, adt.StringKind, adt.IntKind},
		Result: adt.ListKind,
		Func: func(c *internal.CallCtxt) {
			s, sep, n := c.String(0), c.String(1), c.Int(2)
			if c.Do() {
				c.Ret = SplitAfterN(s, sep, n)
			}
		},
	}, {
		Name:   "Split",
		Params: []adt.Kind{adt.StringKind, adt.StringKind},
		Result: adt.ListKind,
		Func: func(c *internal.CallCtxt) {
			s, sep := c.String(0), c.String(1)
			if c.Do() {
				c.Ret = Split(s, sep)
			}
		},
	}, {
		Name:   "SplitAfter",
		Params: []adt.Kind{adt.StringKind, adt.StringKind},
		Result: adt.ListKind,
		Func: func(c *internal.CallCtxt) {
			s, sep := c.String(0), c.String(1)
			if c.Do() {
				c.Ret = SplitAfter(s, sep)
			}
		},
	}, {
		Name:   "Fields",
		Params: []adt.Kind{adt.StringKind},
		Result: adt.ListKind,
		Func: func(c *internal.CallCtxt) {
			s := c.String(0)
			if c.Do() {
				c.Ret = Fields(s)
			}
		},
	}, {
		Name:   "Join",
		Params: []adt.Kind{adt.ListKind, adt.StringKind},
		Result: adt.StringKind,
		Func: func(c *internal.CallCtxt) {
			elems, sep := c.StringList(0), c.String(1)
			if c.Do() {
				c.Ret = Join(elems, sep)
			}
		},
	}, {
		Name:   "HasPrefix",
		Params: []adt.Kind{adt.StringKind, adt.StringKind},
		Result: adt.BoolKind,
		Func: func(c *internal.CallCtxt) {
			s, prefix := c.String(0), c.String(1)
			if c.Do() {
				c.Ret = HasPrefix(s, prefix)
			}
		},
	}, {
		Name:   "HasSuffix",
		Params: []adt.Kind{adt.StringKind, adt.StringKind},
		Result: adt.BoolKind,
		Func: func(c *internal.CallCtxt) {
			s, suffix := c.String(0), c.String(1)
			if c.Do() {
				c.Ret = HasSuffix(s, suffix)
			}
		},
	}, {
		Name:   "Repeat",
		Params: []adt.Kind{adt.StringKind, adt.IntKind},
		Result: adt.StringKind,
		Func: func(c *internal.CallCtxt) {
			s, count := c.String(0), c.Int(1)
			if c.Do() {
				c.Ret = Repeat(s, count)
			}
		},
	}, {
		Name:   "ToUpper",
		Params: []adt.Kind{adt.StringKind},
		Result: adt.StringKind,
		Func: func(c *internal.CallCtxt) {
			s := c.String(0)
			if c.Do() {
				c.Ret = ToUpper(s)
			}
		},
	}, {
		Name:   "ToLower",
		Params: []adt.Kind{adt.StringKind},
		Result: adt.StringKind,
		Func: func(c *internal.CallCtxt) {
			s := c.String(0)
			if c.Do() {
				c.Ret = ToLower(s)
			}
		},
	}, {
		Name:   "Trim",
		Params: []adt.Kind{adt.StringKind, adt.StringKind},
		Result: adt.StringKind,
		Func: func(c *internal.CallCtxt) {
			s, cutset := c.String(0), c.String(1)
			if c.Do() {
				c.Ret = Trim(s, cutset)
			}
		},
	}, {
		Name:   "TrimLeft",
		Params: []adt.Kind{adt.StringKind, adt.StringKind},
		Result: adt.StringKind,
		Func: func(c *internal.CallCtxt) {
			s, cutset := c.String(0), c.String(1)
			if c.Do() {
				c.Ret = TrimLeft(s, cutset)
			}
		},
	}, {
		Name:   "TrimRight",
		Params: []adt.Kind{adt.StringKind, adt.StringKind},
		Result: adt.StringKind,
		Func: func(c *internal.CallCtxt) {
			s, cutset := c.String(0), c.String(1)
			if c.Do() {
				c.Ret = TrimRight(s, cutset)
			}
		},
	}, {
		Name:   "TrimSpace",
		Params: []adt.Kind{adt.StringKind},
		Result: adt.StringKind,
		Func: func(c *internal.CallCtxt) {
			s := c.String(0)
			if c.Do() {
				c.Ret = TrimSpace(s)
			}
		},
	}, {
		Name:   "TrimPrefix",
		Params: []adt.Kind{adt.StringKind, adt.StringKind},
		Result: adt.StringKind,
		Func: func(c *internal.CallCtxt) {
			s, prefix := c.String(0), c.String(1)
			if c.Do() {
				c.Ret = TrimPrefix(s, prefix)
			}
		},
	}, {
		Name:   "TrimSuffix",
		Params: []adt.Kind{adt.StringKind, adt.StringKind},
		Result: adt.StringKind,
		Func: func(c *internal.CallCtxt) {
			s, suffix := c.String(0), c.String(1)
			if c.Do() {
				c.Ret = TrimSuffix(s, suffix)
			}
		},
	}, {
		Name:   "Replace",
		Params: []adt.Kind{adt.StringKind, adt.StringKind, adt.StringKind, adt.IntKind},
		Result: adt.StringKind,
		Func: func(c *internal.CallCtxt) {
			s, old, new, n := c.String(0), c.String(1), c.String(2), c.Int(3)
			if c.Do() {
				c.Ret = Replace(s, old, new, n)
			}
		},
	}, {
		Name:   "Index",
		Params: []adt.Kind{adt.StringKind, adt.StringKind},
		Result: adt.IntKind,
		Func: func(c *internal.CallCtxt) {
			s, substr := c.String(0), c.String(1)
			if c.Do() {
				c.Ret = Index(s, substr)
			}
		},
	}},
}
