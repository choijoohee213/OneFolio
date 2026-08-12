// Package master 는 KIS 종목마스터에서 추린 상장종목 표를 제공한다.
// 잔고파일에는 종목코드가 없고 종목명만 있어서, 이 표가 국내/해외와
// ETF/개별주를 확정하는 유일한 근거다. 데이터 갱신은 cmd/genmaster 로 한다.
package master

import (
	"bufio"
	"bytes"
	"compress/gzip"
	_ "embed"
	"fmt"
	"strings"
)

type Kind string

const (
	DomesticStock Kind = "DS"
	DomesticETF   Kind = "DE"
	ForeignStock  Kind = "FS"
	ForeignETF    Kind = "FE"
)

func (k Kind) IsETF() bool     { return k == DomesticETF || k == ForeignETF }
func (k Kind) IsForeign() bool { return k == ForeignStock || k == ForeignETF }

type Listing struct {
	Code string
	Kind Kind
}

type Entry struct {
	Code string `json:"code"`
	Name string `json:"name"`
	Kind Kind   `json:"kind"`
}

type Table struct {
	byName map[string]Listing // normalized name → listing
	byCode map[string]Entry   // uppercase code → entry
	entries []Entry
}

//go:embed listings.tsv.gz
var listings []byte

func Load() (*Table, error) {
	reader, err := gzip.NewReader(bytes.NewReader(listings))
	if err != nil {
		return nil, fmt.Errorf("종목마스터 압축 해제 실패: %w", err)
	}
	defer reader.Close()

	t := &Table{
		byName: make(map[string]Listing),
		byCode: make(map[string]Entry),
	}
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		parts := strings.SplitN(scanner.Text(), "\t", 3)
		if len(parts) < 3 {
			continue
		}
		code, name, kind := parts[0], parts[1], Kind(parts[2])

		t.byName[normalize(name)] = Listing{Code: code, Kind: kind}
		entry := Entry{Code: code, Name: name, Kind: kind}
		if code != "" {
			t.byCode[strings.ToUpper(code)] = entry
		}
		t.entries = append(t.entries, entry)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("종목마스터 읽기 실패: %w", err)
	}
	return t, nil
}

func Empty() *Table {
	return &Table{byName: make(map[string]Listing), byCode: make(map[string]Entry)}
}

func (t *Table) Len() int { return len(t.entries) }

func (t *Table) Lookup(name string) (Listing, bool) {
	listing, ok := t.byName[normalize(name)]
	return listing, ok
}

func (t *Table) LookupByCode(code string) (Entry, bool) {
	entry, ok := t.byCode[strings.ToUpper(code)]
	return entry, ok
}

// Search 는 질의어로 종목을 찾는다. 이름 접두사 매칭을 우선하고 부분 매칭이 뒤따른다.
// 종목코드로도 검색할 수 있다.
func (t *Table) Search(query string, limit int) []Entry {
	norm := normalize(query)
	if norm == "" {
		return nil
	}
	upper := strings.ToUpper(strings.TrimSpace(query))

	var prefix, substr []Entry
	for _, entry := range t.entries {
		normName := normalize(entry.Name)
		switch {
		case strings.HasPrefix(normName, norm):
			prefix = append(prefix, entry)
		case strings.HasPrefix(strings.ToUpper(entry.Code), upper):
			prefix = append(prefix, entry)
		case strings.Contains(normName, norm):
			substr = append(substr, entry)
		}
		if len(prefix)+len(substr) >= limit*3 {
			break
		}
	}

	results := append(prefix, substr...)
	if len(results) > limit {
		results = results[:limit]
	}
	return results
}

func normalize(name string) string {
	return strings.ToUpper(strings.Join(strings.Fields(name), ""))
}
