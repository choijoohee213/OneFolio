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

var (
	domesticFiles = []string{"kospi_code.mst.zip", "kosdaq_code.mst.zip"}
	overseasFiles = []string{"nasmst.cod.zip", "nysmst.cod.zip", "amsmst.cod.zip"}
)

func main() {
	listings := make(map[string]master.Kind)

	for _, name := range domesticFiles {
		data, err := download(name)
		if err != nil {
			log.Fatalf("%s: %v", name, err)
		}
		readDomestic(data, listings)
	}
	for _, name := range overseasFiles {
		data, err := download(name)
		if err != nil {
			log.Fatalf("%s: %v", name, err)
		}
		readOverseas(data, listings)
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
	nameStart  = 21
	nameEnd    = 61
	groupEnd   = 63
	minLineLen = groupEnd
)

func readDomestic(data []byte, listings map[string]master.Kind) {
	for _, line := range bytes.Split(data, []byte("\n")) {
		line = bytes.TrimRight(line, "\r")
		if len(line) < minLineLen {
			continue
		}

		name := decode(line[nameStart:nameEnd])
		if name == "" {
			continue
		}
		kind := master.DomesticStock
		if group := decode(line[nameEnd:groupEnd]); group == "EF" || group == "EN" {
			kind = master.DomesticETF
		}
		putIfAbsent(listings, name, kind)
	}
}

// 해외 마스터는 탭 구분이다. 7번째가 한글명, 9번째가 증권종류(2:주식, 3:ETP).
const (
	overseasNameColumn = 6
	overseasTypeColumn = 8
	overseasColumns    = 9
)

func readOverseas(data []byte, listings map[string]master.Kind) {
	for _, line := range strings.Split(decode(data), "\n") {
		columns := strings.Split(line, "\t")
		if len(columns) < overseasColumns {
			continue
		}

		name := strings.TrimSpace(columns[overseasNameColumn])
		if name == "" {
			continue
		}
		switch strings.TrimSpace(columns[overseasTypeColumn]) {
		case "2":
			putIfAbsent(listings, name, master.ForeignStock)
		case "3":
			putIfAbsent(listings, name, master.ForeignETF)
		}
	}
}

// 국내 마스터를 먼저 읽으므로 이름이 겹치면 국내가 이긴다.
func putIfAbsent(listings map[string]master.Kind, name string, kind master.Kind) {
	if _, exists := listings[name]; !exists {
		listings[name] = kind
	}
}

func decode(b []byte) string {
	decoded, _, err := transform.Bytes(korean.EUCKR.NewDecoder(), b)
	if err != nil {
		return strings.TrimSpace(string(b))
	}
	return strings.TrimSpace(string(decoded))
}

func write(listings map[string]master.Kind) error {
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
		if _, err := fmt.Fprintf(compressed, "%s\t%s\n", name, listings[name]); err != nil {
			return err
		}
	}
	return nil
}
