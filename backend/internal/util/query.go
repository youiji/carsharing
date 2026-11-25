package util

import (
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
)

type ListQuery struct {
	values url.Values
	v      Validator
}

func NewListQuery(values url.Values) *ListQuery {
	return &ListQuery{values: values}
}

func (l *ListQuery) Search() string {
	return strings.TrimSpace(l.values.Get("q"))
}

func (l *ListQuery) Enum(name string, allowed ...string) *string {
	s := strings.TrimSpace(l.values.Get(name))
	if s == "" {
		return nil
	}
	if !slices.Contains(allowed, s) {
		l.v.Add(name, "допустимые значения: "+strings.Join(allowed, ", "))
		return nil
	}
	return &s
}

func (l *ListQuery) PositiveInt(name string) *int64 {
	s := strings.TrimSpace(l.values.Get(name))
	if s == "" {
		return nil
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil || n <= 0 {
		l.v.Add(name, "должно быть положительным целым числом")
		return nil
	}
	return &n
}

func (l *ListQuery) Sort(allowed ...string) (field string, desc bool) {
	field = allowed[0]
	if s := strings.TrimSpace(l.values.Get("sort")); s != "" {
		if slices.Contains(allowed, s) {
			field = s
		} else {
			l.v.Add("sort", "допустимые значения: "+strings.Join(allowed, ", "))
		}
	}
	switch strings.TrimSpace(l.values.Get("order")) {
	case "", "asc":
	case "desc":
		desc = true
	default:
		l.v.Add("order", "допустимые значения: asc, desc")
	}
	return field, desc
}

func (l *ListQuery) Err() error { return l.v.Err() }

func LikeContains(s string) string {
	r := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return "%" + r.Replace(s) + "%"
}

func PathID(r *http.Request, name string) (int64, error) {
	id, err := strconv.ParseInt(r.PathValue(name), 10, 64)
	if err != nil || id <= 0 {
		return 0, ErrNotFound
	}
	return id, nil
}
