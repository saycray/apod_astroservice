package handler

import (
	"astro"
	"astro/pkg/consts"
	srvc "astro/pkg/service"
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	_ "github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"

	"github.com/gorilla/mux"
)

type Handler struct {
	services *srvc.Service
}

func NewHandler(services *srvc.Service) *Handler {
	return &Handler{services}
}

func (h *Handler) InitRoutes() *mux.Router {

	router := mux.NewRouter()

	// здесь определён порядок использования эндпоинтов
	// первый созраняет пикчу и запись о ней, второй смотрит в базу и отдаёт имена доступных файлов
	// а третий отдаёт файл по имени
	router.HandleFunc("/v1/picday", h.TodaysPicture).Methods(http.MethodGet)
	router.HandleFunc("/v1/stored", h.PicturesFromStorage).Methods(http.MethodGet)
	router.HandleFunc("/v1/storage", h.fileSender)

	return router
}

// функция ищет доступные пикчи
func (h *Handler) PicturesFromStorage(w http.ResponseWriter, r *http.Request) {

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	date := getTimeParam(r, consts.ParamDate)
	start, end := getTimeParam(r, consts.ParamStartDate), getTimeParam(r, consts.ParamEndDate)

	// происходит выбор параметров для поиска
	var data []astro.AstroModel
	if !date.IsZero() {

		model, err := h.services.GetByDate(ctx, date)
		if err != nil {
			sendResponse(w, http.StatusBadRequest, err.Error(), nil)
			return
		}

		data = make([]astro.AstroModel, 1)
		data = append(data, *model)

	} else if !start.IsZero() && !end.IsZero() {

		model, err := h.services.GetByDateRange(ctx, start, end)
		if err != nil {
			logrus.Infof("db err %q", err)
			sendResponse(w, http.StatusBadRequest, err.Error(), nil)
			return
		}

		data = make([]astro.AstroModel, len(model), len(model))
		data = append(data, model...)

	} else {
		sendResponse(w, http.StatusBadRequest, "invalid request params", nil)
		return
	}

	// здесь формируется ответ с массивом урлов на пикчи
	var list []string
	if len(data) != 0 {
		for _, v := range data {

			u, _ := chooseURL(&v)
			res, err := getPicExtension(u)
			if err != nil {
				logrus.Errorf("error while extracting file extension: %q", err)
				continue
			}

			t, err := time.Parse("2006-01-02T15:04:05Z", v.Date)
			if err != nil {
				logrus.Errorf("error while formatting time: %q", err)
				continue
			}

			list = append(list, fmt.Sprintf(`%s:%s/v1/storage?name=%s.%s`, os.Getenv("HOST"), os.Getenv("PORT"), t.Format("2006-01-02"), res))
		}
	}

	sendResponse(w, http.StatusOK, "ok", list)
}

// функция получения пикчи
// сначала смотрит в базу и если есть смысл скачаивает пикчу
func (h *Handler) TodaysPicture(w http.ResponseWriter, r *http.Request) {

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	pc, err := h.services.GetByDate(ctx, time.Now())
	if err != nil {
		logrus.Errorf("Error while checking if data exists: %q", err)
		sendResponse(w, http.StatusInternalServerError, err.Error(), nil)
		return
	}

	if pc != nil {
		u, _ := chooseURL(pc)
		sendResponse(w, http.StatusOK, "already stored", []string{u})
		return
	}

	// я определил эти параметры как необходимые
	// ключ нужен чтобы APOD принял запрос
	// дата это значение по которому нужно искать(сегодня)
	// качество пикчи
	// превью нужно чтобы не сохранять видео
	params := map[string]string{
		consts.ApiKey:      os.Getenv(consts.Token),              // ключ апи для получения инфы от сервиса
		consts.ParamDate:   time.Now().Format(consts.TimeFormat), // дата
		consts.ParamHd:     consts.True,                          // по дефолту запрашивает пикчу в норм качестве
		consts.ParamThumbs: consts.True,                          // заставка для видео, отбрасывается при media_type=image
	}

	// скачивание пикчи
	picture, err := downloadApodPicture(ctx, os.Getenv(consts.AstroURL), params)
	if err != nil {
		logrus.Errorf("Error while downloading picture: %q", err)
		sendResponse(w, http.StatusInternalServerError, err.Error(), nil)
		return
	}

	// сохранение записей о пикче в базе
	n, err := h.services.InsertOne(ctx, picture)
	if err != nil || n != 1 {
		logrus.Errorf("Error while saving data to db %d/1, err : %q", n, err)
		sendResponse(w, http.StatusInternalServerError, err.Error(), nil)
		return
	}

	// сохранение пикчи на диске
	if err := storePicture("pictures", picture); err != nil {
		logrus.Errorf("Error while saving picture to fs: %q", err)
		sendResponse(w, http.StatusInternalServerError, err.Error(), nil)

		// здесь происходит удаление записи из бд если пикча сохранилась с ошибкой
		n, err := h.services.DeleteByDate(ctx, time.Now())
		if err != nil || n != 1 {
			// здесь нечего отсылать, если ошибка в этом месте, значит всё плохо
			logrus.Errorf("Error while removing unsaved picture metadata: %q, %d/1", err, n)
		}

		return
	}

	// выбирает урл и возвращает источник загрузки
	u, _ := chooseURL(picture)
	sendResponse(w, http.StatusOK, "successfully downloaded", []string{u})
}

// функция которая по имени отдаёт пикчу
func (h *Handler) fileSender(w http.ResponseWriter, r *http.Request) {

	fn := getStringParam(r, consts.Filename)

	if fn == "" {
		sendResponse(w, http.StatusNotFound, "no filename", nil)
		return
	}

	// своеобразный способ проверки имени файла из запроса
	ext, err := getPicExtension(fn)
	if err != nil {
		sendResponse(w, http.StatusNotFound, err.Error(), nil)
		return
	}

	// получение пикчи с диска
	fl, err := os.ReadFile("/go/pictures/" + fn)
	if err != nil {
		sendResponse(w, http.StatusInternalServerError, err.Error(), nil)
		return
	}

	w.Header().Add("Content-Type", "image/"+ext)
	w.WriteHeader(http.StatusOK)
	w.Write(fl)
}
