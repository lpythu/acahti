package page

import (
	"net/http"
	"testing"
)

func TestParse(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/x?page=3&page_size=10", nil)
	q := Parse(req)
	if q.Page != 3 || q.Size != 10 {
		t.Fatalf("got %+v", q)
	}
	req, _ = http.NewRequest(http.MethodGet, "/x?page=-1&page_size=999", nil)
	q = Parse(req)
	if q.Page != 1 || q.Size != MaxSize {
		t.Fatalf("clamp got %+v", q)
	}
}

func TestClipTake(t *testing.T) {
	q := Query{Page: 1, Size: 2}
	got := Clip([]int{1, 2, 3}, q)
	if !got.HasMore || len(got.Items) != 2 || got.Items[1] != 2 {
		t.Fatalf("clip %+v", got)
	}
	got = Take([]int{1, 2, 3, 4, 5}, Query{Page: 2, Size: 2})
	if !got.HasMore || len(got.Items) != 2 || got.Items[0] != 3 {
		t.Fatalf("take %+v", got)
	}
	got = Take([]int{1, 2}, Query{Page: 2, Size: 2})
	if got.HasMore || len(got.Items) != 0 {
		t.Fatalf("empty page %+v", got)
	}
}

func TestWalk(t *testing.T) {
	pages := [][]int{{1, 2}, {3}}
	n := 0
	all, err := Walk(func(q Query) (Result[int], error) {
		n++
		if q.Page > len(pages) {
			return Of([]int{}, q, false), nil
		}
		items := pages[q.Page-1]
		return Of(items, q, q.Page < len(pages)), nil
	})
	if err != nil || n != 2 || len(all) != 3 || all[2] != 3 {
		t.Fatalf("walk n=%d all=%v err=%v", n, all, err)
	}
}
