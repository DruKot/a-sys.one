package server

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/valyala/fasthttp"
)

type Server struct {
	uploadDir string
	host      string
	port      int
}

func NewServer(host string, port int, uploadDir string) *Server {
	return &Server{
		host:      host,
		port:      port,
		uploadDir: uploadDir,
	}
}

func (s *Server) Listen() {
	addr := fmt.Sprintf("%s:%d", s.host, s.port)
	if err := fasthttp.ListenAndServe(addr, s.handler); err != nil {
		fmt.Printf("Error starting server: %v\n", err)
	}
}

func (s *Server) handler(ctx *fasthttp.RequestCtx) {
	file, err := os.OpenFile(s.uploadDir+"\\temp.bin", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	// Получаем boundary из Content-Type
	boundary := extractBoundary(string(ctx.Request.Header.ContentType()))
	if boundary == "" {
		log.Fatalf("Boundary не найден, код %d", fasthttp.StatusBadRequest)
		return
	}

	body := ctx.Request.Body()
	if len(body) == 0 {
		ctx.Error("Empty body", fasthttp.StatusBadRequest)
		return
	}

	// Разбираем multipart
	if err := processMultipart(body, boundary, file); err != nil {
		log.Fatalf("%d %v", fasthttp.StatusInternalServerError, err)
		return
	}

	ctx.SetStatusCode(fasthttp.StatusOK)
	fmt.Fprintf(ctx, "Чанк получен")
}

// extractBoundary извлекает boundary из Content-Type
func extractBoundary(contentType string) string {
	parts := strings.Split(contentType, ";")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "boundary=") {
			return strings.TrimPrefix(part, "boundary=")
		}
	}
	return ""
}

// processMultipart разбирает multipart данные и дописывает части в файл
func processMultipart(data []byte, boundary string, file *os.File) error {
	// Границы в multipart
	boundaryDelimiter := "--" + boundary
	boundaryEnd := boundaryDelimiter + "--"

	// Ищем начало первой части
	startIdx := bytes.Index(data, []byte(boundaryDelimiter))
	if startIdx == -1 {
		return fmt.Errorf("no boundary found")
	}

	// Позиция после первого boundary
	pos := startIdx + len(boundaryDelimiter)

	for {
		// Пропускаем \r\n после boundary
		if pos+2 <= len(data) && data[pos] == '\r' && data[pos+1] == '\n' {
			pos += 2
		}

		// Ищем заголовки части (до пустой строки)
		headersEnd := bytes.Index(data[pos:], []byte("\r\n\r\n"))
		if headersEnd == -1 {
			break
		}

		// Позиция после заголовков (начало данных)
		dataStart := pos + headersEnd + 4

		// Ищем конец этой части (следующий boundary)
		nextBoundary := bytes.Index(data[dataStart:], []byte(boundaryDelimiter))
		if nextBoundary == -1 {
			break
		}

		// Данные части (до \r\n перед следующим boundary)
		partData := data[dataStart : dataStart+nextBoundary]

		// Убираем конечные \r\n если есть
		if len(partData) >= 2 && partData[len(partData)-2] == '\r' && partData[len(partData)-1] == '\n' {
			partData = partData[:len(partData)-2]
		}

		// Дописываем данные в файл
		if len(partData) > 0 {
			if _, err := file.Write(partData); err != nil {
				return fmt.Errorf("failed to write to file: %w", err)
			}
			log.Printf("Wrote %d bytes to file", len(partData))
		}

		// Проверяем, не конец ли это
		nextPos := dataStart + nextBoundary
		if nextPos+len(boundaryEnd) <= len(data) &&
			string(data[nextPos:nextPos+len(boundaryEnd)]) == boundaryEnd {
			break
		}

		// Переходим к следующей части
		pos = nextPos + len(boundaryDelimiter)
	}

	return nil
}
