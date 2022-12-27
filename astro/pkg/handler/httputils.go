package handler

import (
	"astro"
	"astro/pkg/consts"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/jmoiron/sqlx"

	"github.com/sirupsen/logrus"
)

// вытягивает время из запроса
func getTimeParam(r *http.Request, name string) time.Time {

	if r == nil {
		return time.Time{}
	}

	t, err := time.Parse(consts.TimeFormat, r.URL.Query().Get(name))
	if err != nil {
		return time.Time{}
	}

	return t
}

// вытягивает строку из запроса, добавил эту функцию чтобы было +- в одном стиле
func getStringParam(r *http.Request, name string) string {

	if r == nil {
		return ""
	}

	return r.URL.Query().Get(name)
}

// здесь происходит вся магия получения пикчи
// сначала формируется строка запроса со всеми параметрами, которые я выделил как неодходимые
// APOD возвращает описчание пикчи из этих данных выбирается урла для скачивания
// в приоритете беру в HD качестве и скачиваю
func downloadApodPicture(ctx context.Context, baseUrl string, params map[string]string) (*astro.AstroModel, error) {

	req, err := makeRequest(baseUrl, params)
	if err != nil {
		return nil, err
	}

	logrus.Infof("request info: %s", req)

	md, err := getMetadata(ctx, req)
	if err != nil {
		return nil, err
	}

	logrus.Infof("metadata info: %v", md)

	url, err := chooseURL(md)
	if err != nil {
		return nil, err
	}

	logrus.Infof("url info: %v", url)

	raw, err := downloadPicture(ctx, url)
	if err != nil {
		return nil, err
	}

	md.RAW = raw
	return md, nil
}

// конструктор для формирования строки запроса
func makeRequest(baseUrl string, params map[string]string) (string, error) {
	ur, err := url.Parse(baseUrl)
	if err != nil {
		return "", err
	}

	q := ur.Query()
	for k, v := range params {
		q.Set(k, v)
	}

	ur.RawQuery = q.Encode()
	return ur.String(), nil
}

// функция для получения описания пикчи
func getMetadata(ctx context.Context, u string) (*astro.AstroModel, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var metadata astro.AstroModel
	if err := json.Unmarshal(body, &metadata); err != nil {
		return nil, err
	}

	return &metadata, nil
}

// здесь спрятана логика получения урлы для скачивания
// Почему так: старые записи не имеют HD качества, поэтому приходит только ссылка на скачивания в обычном качестве
// некоторые пикчи дня - видео, их можно было бы сохранить, но я выбрал сохранять превью, для чего и использую в конструкторе параметр thumbnail
func chooseURL(m *astro.AstroModel) (string, error) {

	var url = ""
	var err = error(nil)

	switch {
	case m.MediaType == "image" && m.HDURL != "":
		url = m.HDURL
	case m.MediaType == "image" && m.URL != "":
		url = m.URL
	case m.MediaType == "video" && m.ThumbURL != "":
		url = m.ThumbURL
	default:
		logrus.Warnf("Unexpected case: object: %v, ", m)
		err = errors.New("invalid url")
	}

	return url, err
}

// здесь ничего интересного, обычная загрузка байтов
func downloadPicture(ctx context.Context, url string) ([]byte, error) {

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

// описание ответа сервера
type Response struct {
	Message string   `json:"message"`
	Urls    []string `json:"urls"`
}

// тут пикча созраняется на диск
func storePicture(path string, md *astro.AstroModel) error {

	ur, err := chooseURL(md)
	if err != nil {
		return err
	}

	res, err := getPicExtension(ur)
	if err != nil {
		return err
	}

	fullName := filepath.Join(path, md.Date+"."+res)

	_, err = os.Stat(fullName)
	if errors.Is(err, os.ErrExist) {
		return nil
	}

	f, err := os.Create(fullName)
	if err != nil {
		return err
	}

	logrus.Info("file stored to ", f.Name())

	defer f.Close()

	if err := f.Chmod(os.FileMode(0777)); err != nil {
		return err
	}

	bytes, err := f.Write(md.RAW)
	if err != nil || bytes != len(md.RAW) {
		return fmt.Errorf("fs error: %q, bytes: %d/%d", err, len(md.RAW), bytes)
	}

	return nil
}

// чтобы не плодить один и тот же код, я вынес ответ от сервера в одну функцию
func sendResponse(w http.ResponseWriter, status int, msg string, urls []string) {
	w.WriteHeader(status)
	w.Header().Add("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(Response{Message: msg, Urls: urls}); err != nil {
		logrus.Errorf("error while sending response %q", err)
	}
}

// вытягивает и аповеряет формат файла
func getPicExtension(str string) (string, error) {

	if len(str) == 0 {
		return "", errors.New("empty string")
	}

	list := strings.Split(str, ".")
	if len(list) == 0 {
		return "", errors.New("no file extension")
	}

	var found bool
	for _, v := range []string{"jpg", "jpeg", "png"} {
		if list[len(list)-1] == v {
			found = true
			break
		}
	}

	if !found {
		return "", errors.New("no file extension")
	}

	return list[len(list)-1], nil
}
