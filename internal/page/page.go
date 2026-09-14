package page

import (
	"net/http"
	"strconv"
)

const (
	DefaultSize = 20
	MaxSize     = 50
	MaxWalk     = 40
)

type Query struct {
	Page int
	Size int
}

func (q Query) Norm() Query {
	if q.Page <= 0 {
		q.Page = 1
	}
	if q.Size <= 0 {
		q.Size = DefaultSize
	}
	if q.Size > MaxSize {
		q.Size = MaxSize
	}
	return q
}

func (q Query) LimitPlus() int {
	return q.Norm().Size + 1
}

func Parse(r *http.Request) Query {
	return From(r.URL.Query().Get("page"), r.URL.Query().Get("page_size"))
}

func From(page, size string) Query {
	q := Query{Page: 1, Size: DefaultSize}
	if n, err := strconv.Atoi(page); err == nil {
		q.Page = n
	}
	if n, err := strconv.Atoi(size); err == nil {
		q.Size = n
	}
	return q.Norm()
}

func FromInts(page, size int) Query {
	return Query{Page: page, Size: size}.Norm()
}

type Result[T any] struct {
	Items   []T  `json:"items"`
	Page    int  `json:"page"`
	Size    int  `json:"page_size"`
	HasMore bool `json:"has_more"`
}

func Of[T any](items []T, q Query, hasMore bool) Result[T] {
	if items == nil {
		items = []T{}
	}
	q = q.Norm()
	return Result[T]{Items: items, Page: q.Page, Size: q.Size, HasMore: hasMore}
}

func Clip[T any](items []T, q Query) Result[T] {
	q = q.Norm()
	if items == nil {
		items = []T{}
	}
	more := len(items) > q.Size
	if more {
		items = items[:q.Size]
	}
	return Of(items, q, more)
}

func Take[T any](all []T, q Query) Result[T] {
	q = q.Norm()
	if all == nil {
		all = []T{}
	}
	start := (q.Page - 1) * q.Size
	if start >= len(all) {
		return Of([]T{}, q, false)
	}
	end := start + q.Size
	more := end < len(all)
	if end > len(all) {
		end = len(all)
	}
	return Of(all[start:end], q, more)
}

func Walk[T any](fn func(Query) (Result[T], error)) ([]T, error) {
	q := Query{Page: 1, Size: MaxSize}
	var all []T
	for q.Page <= MaxWalk {
		res, err := fn(q)
		if err != nil {
			return nil, err
		}
		all = append(all, res.Items...)
		if !res.HasMore {
			break
		}
		q.Page++
	}
	if all == nil {
		all = []T{}
	}
	return all, nil
}
