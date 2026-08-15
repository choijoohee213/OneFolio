// server 는 잔고파일을 받아 자산배분 집계를 돌려주는 HTTP 서버다.
//
// 환경변수:
//
//	PORT             수신 포트 (기본 8080). Render 가 주입한다.
//	ALLOWED_ORIGINS  CORS 허용 오리진, 쉼표로 구분 (비우면 모두 허용)
package main

import (
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/choijoohee213/OneFolio/backend/internal/api"
	"github.com/choijoohee213/OneFolio/backend/internal/master"
	"github.com/choijoohee213/OneFolio/backend/internal/ocr"
)

const (
	defaultPort  = "8080"
	readTimeout  = 30 * time.Second
	writeTimeout = 60 * time.Second
	idleTimeout  = 120 * time.Second
)

func main() {
	listings, err := master.Load()
	if err != nil {
		log.Fatalf("종목마스터 로드 실패: %v", err)
	}

	var ocrClient *ocr.Client
	if keys := os.Getenv("GEMINI_API_KEY"); keys != "" {
		c, err := ocr.NewClient(keys)
		if err != nil {
			log.Fatalf("OCR 클라이언트 초기화 실패: %v", err)
		}
		ocrClient = c
		log.Printf("OCR 활성화 (Gemini, 키 %d개)", c.KeyCount())
	}

	mux := http.NewServeMux()
	api.New(listings, ocrClient).Register(mux)

	address := ":" + env("PORT", defaultPort)
	server := &http.Server{
		Addr:         address,
		Handler:      api.CORS(strings.Split(os.Getenv("ALLOWED_ORIGINS"), ","), mux),
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
		IdleTimeout:  idleTimeout,
	}

	log.Printf("listening on %s (종목마스터 %d건)", address, listings.Len())
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
