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

type Table map[string]Kind

//go:embed listings.tsv.gz
var listings []byte

func Load() (Table, error) {
	reader, err := gzip.NewReader(bytes.NewReader(listings))
	if err != nil {
		return nil, fmt.Errorf("종목마스터 압축 해제 실패: %w", err)
	}
	defer reader.Close()

	table := make(Table)
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		name, kind, ok := strings.Cut(scanner.Text(), "\t")
		if !ok {
			continue
		}
		table[normalize(name)] = Kind(kind)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("종목마스터 읽기 실패: %w", err)
	}
	return table, nil
}

func (t Table) Lookup(name string) (Kind, bool) {
	kind, ok := t[normalize(name)]
	return kind, ok
}

// 증권사마다 띄어쓰기와 대소문자가 달라서 그것만 없애고 맞춘다.
func normalize(name string) string {
	return strings.ToUpper(strings.Join(strings.Fields(name), ""))
}
