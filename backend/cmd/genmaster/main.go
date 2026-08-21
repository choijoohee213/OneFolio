// genmaster 는 KIS 종목마스터를 받아 분류에 필요한 컬럼만 추려
// internal/master/listings.tsv.gz 를 다시 만든다. 신규 상장이 반영되지 않을 때 돌린다.
//
//	go run ./cmd/genmaster
package main

import (
	"archive/zip"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"

	"golang.org/x/text/encoding/korean"
	"golang.org/x/text/transform"

	"github.com/choijoohee213/OneFolio/backend/internal/master"
)

const (
	baseURL = "https://new.real.download.dws.co.kr/common/master/"
	outPath = "internal/master/listings.tsv.gz"
)

// 시장은 파일별로 갈린다 — 어느 파일에서 읽었는지가 곧 상장 시장이다.
var (
	domesticFiles = map[string]master.Market{
		"kospi_code.mst.zip":  master.KOSPI,
		"kosdaq_code.mst.zip": master.KOSDAQ,
	}
	overseasFiles = map[string]master.Market{
		"nasmst.cod.zip": master.NASDAQ,
		"nysmst.cod.zip": master.NYSE,
		"amsmst.cod.zip": master.AMEX,
	}
)

type listing struct {
	code   string
	kind   master.Kind
	market master.Market
}

func main() {
	listings := make(map[string]listing)

	for _, name := range sortedNames(domesticFiles) {
		data, err := download(name)
		if err != nil {
			log.Fatalf("%s: %v", name, err)
		}
		readDomestic(data, listings, domesticFiles[name])
	}
	for _, name := range sortedNames(overseasFiles) {
		data, err := download(name)
		if err != nil {
			log.Fatalf("%s: %v", name, err)
		}
		readOverseas(data, listings, overseasFiles[name])
	}

	if err := write(listings); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s 생성 완료 (%d종목)\n", outPath, len(listings))
}

func download(name string) ([]byte, error) {
	resp, err := http.Get(baseURL + name)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	archive, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		return nil, err
	}
	if len(archive.File) == 0 {
		return nil, fmt.Errorf("빈 zip")
	}

	entry, err := archive.File[0].Open()
	if err != nil {
		return nil, err
	}
	defer entry.Close()
	return io.ReadAll(entry)
}

// 국내 마스터는 고정길이다. 단축코드 9 + 표준코드 12 + 한글종목명 40 뒤에 증권그룹구분코드가 온다.
const (
	codeEnd    = 9
	nameStart  = 21
	nameEnd    = 61
	groupEnd   = 63
	minLineLen = groupEnd
)

func readDomestic(data []byte, listings map[string]listing, market master.Market) {
	for _, line := range bytes.Split(data, []byte("\n")) {
		line = bytes.TrimRight(line, "\r")
		if len(line) < minLineLen {
			continue
		}

		code := decode(line[0:codeEnd])
		name := decode(line[nameStart:nameEnd])
		if name == "" {
			continue
		}
		kind := master.DomesticStock
		if group := decode(line[nameEnd:groupEnd]); group == "EF" || group == "EN" {
			kind = master.DomesticETF
		}
		putIfAbsent(listings, name, listing{code: code, kind: kind, market: market})
	}
}

// 해외 마스터는 탭 구분이다. 5번째가 티커, 7번째가 한글명, 9번째가 증권종류(2:주식, 3:ETP).
const (
	overseasCodeColumn = 4
	overseasNameColumn = 6
	overseasTypeColumn = 8
	overseasColumns    = 9
)

func readOverseas(data []byte, listings map[string]listing, market master.Market) {
	for _, line := range strings.Split(decode(data), "\n") {
		columns := strings.Split(line, "\t")
		if len(columns) < overseasColumns {
			continue
		}

		code := strings.TrimSpace(columns[overseasCodeColumn])
		name := strings.TrimSpace(columns[overseasNameColumn])
		if name == "" {
			continue
		}
		switch strings.TrimSpace(columns[overseasTypeColumn]) {
		case "2":
			putIfAbsent(listings, name, listing{code: code, kind: master.ForeignStock, market: market})
		case "3":
			putIfAbsent(listings, name, listing{code: code, kind: master.ForeignETF, market: market})
		}
	}
}

// 맵 순회 순서는 매번 달라진다. 같은 종목명이 두 시장에 있으면 먼저 읽은 쪽이
// 이기므로(putIfAbsent), 순서를 고정해야 실행할 때마다 결과가 바뀌지 않는다.
func sortedNames(files map[string]master.Market) []string {
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func putIfAbsent(listings map[string]listing, name string, l listing) {
	if _, exists := listings[name]; !exists {
		listings[name] = l
	}
}

func decode(b []byte) string {
	decoded, _, err := transform.Bytes(korean.EUCKR.NewDecoder(), b)
	if err != nil {
		return strings.TrimSpace(string(b))
	}
	return strings.TrimSpace(string(decoded))
}

func write(listings map[string]listing) error {
	names := make([]string, 0, len(listings))
	for name := range listings {
		names = append(names, name)
	}
	sort.Strings(names)

	file, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer file.Close()

	compressed, err := gzip.NewWriterLevel(file, gzip.BestCompression)
	if err != nil {
		return err
	}
	defer compressed.Close()

	for _, name := range names {
		l := listings[name]
		if _, err := fmt.Fprintf(compressed, "%s\t%s\t%s\t%s\n", l.code, name, l.kind, l.market); err != nil {
			return err
		}
	}
	return nil
}
