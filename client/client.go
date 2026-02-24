package client

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"github.com/valyala/fasthttp"
)

// TODO Переделать на pipe и слать чанки внутри единой multipart формы?
// TODO Хранить статус передачи что бы продолжить при разрыве соединения
// TODO Передавать завершающий запрос что бы переименовать временный файл и сообщить серверу что мы закончили
// TODO Прогрессбар
// TODO Логирование
// TODO Тесты
// TODO Конфиг файл

type Config struct {
	ChunkSize int
}

type Client struct {
	fc  *fasthttp.Client
	cfg Config
}

func New() *Client {
	return &Client{
		cfg: Config{
			ChunkSize: 256,
		},
		fc: &fasthttp.Client{},
	}
}

func (c *Client) SendFile(url string, fileName string) error {
	file, err := os.Open(fileName)
	if err != nil {
		return err
	}
	defer file.Close()

	buf := make([]byte, c.cfg.ChunkSize)
	part := 0

	// Заполняем буфер и сразу отправляем что бы не забивать память
	for {
		var n int
		n, err = file.Read(buf)
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if n > 0 {
			if err = c.sendChunk(buf, part, url, file.Name()); err != nil {
				return err
			}
			part++
		}
	}

	return nil
}

func (c *Client) sendChunk(data []byte, index int, url string, filename string) error {
	body := &bytes.Buffer{}

	// Оформляем заголовок чанка
	fmt.Fprintf(body, "--chunkboundary\r\n")
	fmt.Fprintf(body, "Content-Disposition: form-data; name=\"chunk\"; filename=\"chunk-%d\"\r\n", index)
	fmt.Fprintf(body, "Content-Type: application/octet-stream\r\n\r\n")

	// Пишем тело
	body.Write(data)

	// Оформляем footer
	body.WriteString("\r\n--chunkboundary--\r\n")

	// Создаем HTTP запрос
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	req.SetRequestURI(url)
	req.Header.SetMethod("POST")
	req.Header.SetContentType("multipart/form-data; boundary=chunkboundary")
	req.SetBody(body.Bytes())

	// Выполняем запрос
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	if err := c.fc.Do(req, resp); err != nil {
		return err
	}

	if resp.StatusCode() != fasthttp.StatusOK {
		return fmt.Errorf("Сервер вернул статус %d", resp.StatusCode())
	}

	return nil
}
