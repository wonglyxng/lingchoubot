package handler

import (
	"net/http"
	"net/url"
	"testing"
)

func TestParsePagination_Defaults(t *testing.T) {
	r := &http.Request{URL: &url.URL{RawQuery: ""}}
	limit, offset := parsePagination(r)
	if limit != 20 {
		t.Errorf("default limit: expected 20, got %d", limit)
	}
	if offset != 0 {
		t.Errorf("default offset: expected 0, got %d", offset)
	}
}

func TestParsePagination_CustomValues(t *testing.T) {
	r := &http.Request{URL: &url.URL{RawQuery: "limit=50&offset=10"}}
	limit, offset := parsePagination(r)
	if limit != 50 {
		t.Errorf("expected limit 50, got %d", limit)
	}
	if offset != 10 {
		t.Errorf("expected offset 10, got %d", offset)
	}
}

func TestParsePagination_MaxLimit(t *testing.T) {
	r := &http.Request{URL: &url.URL{RawQuery: "limit=200"}}
	limit, _ := parsePagination(r)
	if limit != 20 {
		t.Errorf("limit > 100 should fall back to default 20, got %d", limit)
	}
}

func TestParsePagination_ZeroLimit(t *testing.T) {
	r := &http.Request{URL: &url.URL{RawQuery: "limit=0"}}
	limit, _ := parsePagination(r)
	if limit != 20 {
		t.Errorf("limit=0 should fall back to default 20, got %d", limit)
	}
}

func TestParsePagination_NegativeLimit(t *testing.T) {
	r := &http.Request{URL: &url.URL{RawQuery: "limit=-5"}}
	limit, _ := parsePagination(r)
	if limit != 20 {
		t.Errorf("negative limit should fall back to default 20, got %d", limit)
	}
}

func TestParsePagination_NegativeOffset(t *testing.T) {
	r := &http.Request{URL: &url.URL{RawQuery: "offset=-1"}}
	_, offset := parsePagination(r)
	if offset != 0 {
		t.Errorf("negative offset should fall back to default 0, got %d", offset)
	}
}

func TestParsePagination_NonNumeric(t *testing.T) {
	r := &http.Request{URL: &url.URL{RawQuery: "limit=abc&offset=xyz"}}
	limit, offset := parsePagination(r)
	if limit != 20 {
		t.Errorf("non-numeric limit should fall back to default 20, got %d", limit)
	}
	if offset != 0 {
		t.Errorf("non-numeric offset should fall back to default 0, got %d", offset)
	}
}

func TestParsePagination_BoundaryLimit(t *testing.T) {
	r := &http.Request{URL: &url.URL{RawQuery: "limit=100"}}
	limit, _ := parsePagination(r)
	if limit != 100 {
		t.Errorf("limit=100 should be accepted, got %d", limit)
	}
}

func TestParsePagination_LimitOne(t *testing.T) {
	r := &http.Request{URL: &url.URL{RawQuery: "limit=1"}}
	limit, _ := parsePagination(r)
	if limit != 1 {
		t.Errorf("limit=1 should be accepted, got %d", limit)
	}
}

func TestParsePagination_LargeOffset(t *testing.T) {
	r := &http.Request{URL: &url.URL{RawQuery: "offset=99999"}}
	_, offset := parsePagination(r)
	if offset != 99999 {
		t.Errorf("large offset should be accepted, got %d", offset)
	}
}
