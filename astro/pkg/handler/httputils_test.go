package handler

import (
	"astro"
	"astro/pkg/consts"
	"context"
	"crypto/sha256"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/lib/pq"

	"github.com/stretchr/testify/require"
)

func TestGetTimeParam(t *testing.T) {

	loc, err := time.LoadLocation("UTC")
	require.NoError(t, err)
	require.NotNil(t, loc)

	tests := []struct {
		name     string
		payload  string
		param    string
		nilReq   bool
		expected time.Time
	}{
		{
			name:     "Get valid date",
			payload:  "https://hehe.org/hehe?date=2013-09-30",
			param:    "date",
			expected: time.Date(2013, 9, 30, 0, 0, 0, 0, loc),
		}, {
			name:     "Get not valid date",
			payload:  "https://hehe.org/hehe?date=2013-09-300",
			param:    "date",
			expected: time.Date(1, 1, 1, 0, 0, 0, 0, loc),
		}, {
			name:     "Get empty date",
			payload:  "https://hehe.org/hehe?date=",
			param:    "date",
			expected: time.Date(1, 1, 1, 0, 0, 0, 0, loc),
		}, {
			name:     "Get date in another month/day format",
			payload:  "https://hehe.org/hehe?date=2010-9-3",
			param:    "date",
			expected: time.Date(1, 1, 1, 0, 0, 0, 0, loc),
		}, {
			name:     "Nil req",
			payload:  "https://hehe.org/hehe?date=2010-9-3",
			param:    "date",
			expected: time.Time{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			req, err := http.NewRequest(http.MethodGet, tt.payload, nil)
			require.NoError(t, err)

			if tt.nilReq {
				req = nil
			}

			actual := getTimeParam(req, tt.param)
			require.Equal(t, tt.expected, actual)
		})
	}
}

func TestGetStringParam(t *testing.T) {

	tests := []struct {
		name     string
		payload  string
		param    string
		nilReq   bool
		expected string
	}{
		{
			name:     "Get valid value",
			payload:  "https://hehe.org/hehe?date=2013-09-30",
			param:    "date",
			expected: "2013-09-30",
		}, {
			name:     "Get empty string",
			payload:  "https://hehe.org/hehe?date=",
			param:    "date",
			expected: "",
		}, {
			name:     "Nil req",
			payload:  "https://hehe.org/hehe?date=",
			param:    "date",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			req, err := http.NewRequest(http.MethodGet, tt.payload, nil)
			require.NoError(t, err)

			if tt.nilReq {
				req = nil
			}

			actual := getStringParam(req, tt.param)
			require.Equal(t, tt.expected, actual)
		})
	}
}

func TestDownloadApodPicture(t *testing.T) {

	tests := []struct {
		name          string
		timeout       time.Duration
		baseUrl       string
		params        map[string]string
		expected      [32]byte //sha256
		expectedError error
	}{
		{
			name:          "bad params",
			baseUrl:       "https://nasa.gov",
			params:        nil,
			timeout:       10 * time.Second,
			expectedError: errors.New("invalid character '<' looking for beginning of value"),
		},
		{
			name:          "bad params",
			baseUrl:       "https://api.nasa.gov/planetary/apod",
			params:        map[string]string{"date": "2022-01-01"},
			timeout:       10 * time.Second,
			expectedError: errors.New("invalid url"),
		},
		{
			name:          "no time",
			baseUrl:       "https://api.nasa.gov/planetary/apod",
			params:        map[string]string{"api_key": os.Getenv(consts.Token), "date": "2022-01-01"},
			timeout:       10 * time.Nanosecond,
			expectedError: errors.New(`Get "https://api.nasa.gov/planetary/apod?api_key=` + os.Getenv(consts.Token) + `&date=2022-01-01": context deadline exceeded`),
		}, {
			name:          "should be ok",
			baseUrl:       "https://api.nasa.gov/planetary/apod",
			params:        map[string]string{"api_key": os.Getenv(consts.Token), "date": "2022-01-01"},
			timeout:       10 * time.Second,
			expected:      [32]uint8{0x2f, 0x44, 0xe9, 0xf2, 0x8f, 0xdd, 0x62, 0x34, 0x4d, 0x7, 0xa6, 0x60, 0xf, 0x87, 0xb1, 0x44, 0xac, 0x29, 0xa8, 0xc2, 0xb7, 0x5e, 0x60, 0x49, 0x1f, 0xed, 0x6a, 0x7a, 0x93, 0xd8, 0x63, 0xc3},
			expectedError: nil,
		},
	}

	for _, tt := range tests {

		ctx, cancel := context.WithTimeout(context.Background(), tt.timeout)
		defer cancel()

		md, err := downloadApodPicture(ctx, tt.baseUrl, tt.params)
		if tt.expectedError == nil {
			require.NoError(t, err)

			//raw - самое последнее полученное значение, если оно есть, значит всё ок
			require.Equal(t, tt.expected, sha256.Sum256(md.RAW))

		} else {
			require.Equal(t, tt.expectedError.Error(), err.Error())
		}
	}
}

func TestDownloadPicture(t *testing.T) {

	tests := []struct {
		name          string
		url           string
		expected      [32]byte // raw represented as sha256(raw)
		expectedError error
	}{
		{
			name:     "ok 2022-01-01 url",
			url:      "https://apod.nasa.gov/apod/image/2201/MoonstripsAnnotatedIG_crop1024.jpg",
			expected: [32]uint8{0x2f, 0x44, 0xe9, 0xf2, 0x8f, 0xdd, 0x62, 0x34, 0x4d, 0x7, 0xa6, 0x60, 0xf, 0x87, 0xb1, 0x44, 0xac, 0x29, 0xa8, 0xc2, 0xb7, 0x5e, 0x60, 0x49, 0x1f, 0xed, 0x6a, 0x7a, 0x93, 0xd8, 0x63, 0xc3},
		}, {
			name:     "ok 2022-01-01 hd url",
			url:      "https://apod.nasa.gov/apod/image/2201/MoonstripsAnnotatedIG.jpg",
			expected: [32]uint8{0xdb, 0xd2, 0x39, 0x9a, 0x27, 0xf1, 0xdd, 0xbd, 0xe5, 0x17, 0x1, 0x3e, 0x53, 0xf0, 0xd0, 0x68, 0x45, 0xfd, 0x1a, 0xae, 0xfe, 0xee, 0xc8, 0x97, 0x94, 0x61, 0x4e, 0x8a, 0x49, 0xc, 0x5b, 0x50},
		}, {
			name:     "ok 2022-02-01 thumb url for video",
			url:      "https://img.youtube.com/vi/c4Xky6tlFyY/0.jpg",
			expected: [32]uint8{0xa6, 0x95, 0xda, 0x3c, 0x2a, 0x12, 0x4e, 0x7e, 0xd, 0x8c, 0x69, 0xe4, 0xf2, 0x24, 0x78, 0x68, 0x79, 0x94, 0xf8, 0xab, 0xaf, 0xce, 0x89, 0xb4, 0x9f, 0x2c, 0xc, 0xb1, 0x2a, 0x47, 0xaf, 0xf},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			bytes, err := downloadPicture(context.Background(), tt.url)
			require.NoError(t, err)

			require.Equal(t, tt.expected, sha256.Sum256(bytes))
		})
	}
}

func TestMakeRequest(t *testing.T) {
	tests := []struct {
		name     string
		baseUrl  string
		params   map[string]string
		expected string
	}{
		{
			name:     "Get url with no params",
			baseUrl:  "https://hehe.org/hehe",
			expected: "https://hehe.org/hehe",
		},
		{
			name:     "Get url with one param",
			baseUrl:  "https://hehe.org/hehe",
			params:   map[string]string{"count": "1"},
			expected: "https://hehe.org/hehe?count=1",
		}, {
			name:     "Get url with several params",
			baseUrl:  "https://hehe.org/hehe",
			params:   map[string]string{"count": "1", "not": "hehe"},
			expected: "https://hehe.org/hehe?count=1&not=hehe",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			actual, err := makeRequest(tt.baseUrl, tt.params)

			require.NoError(t, err)
			require.Equal(t, tt.expected, actual)
		})
	}
}

func TestGetMetadata(t *testing.T) {

	tests := []struct {
		name          string
		timeout       time.Duration
		url           string
		expected      *astro.AstroModel
		expectedError error
	}{
		{
			name:    "ok",
			timeout: 10 * time.Second,
			url:     `https://api.nasa.gov/planetary/apod?api_key=` + os.Getenv(consts.Token) + `&date=2022-01-01`,
			expected: &astro.AstroModel{
				Copyright:   "Soumyadeep Mukherjee",
				Date:        "2022-01-01",
				Explanation: "very Full Moon of 2021 shines in this year-spanning astrophoto project, a composite portrait of the familiar lunar nearside at each brightest lunar phase. Arranged by moonth, the year progresses in stripes beginning at the top. Taken with the same camera and lens the stripes are from Full Moon images all combined at the same pixel scale. The stripes still look mismatched, but they show that the Full Moon's angular size changes throughout the year depending on its distance from Kolkata, India, planet Earth. The calendar month, a full moon name, distance in kilometers, and angular size is indicated for each stripe. Angular size is given in minutes of arc corresponding to 1/60th of a degree. The largest Full Moon is near a perigee or closest approach in May. The smallest is near an apogee, the most distant Full Moon in December. Of course the full moons of May and November also slid into Earth's shadow during 2021's two lunar eclipses.",
				MediaType:   "image",
				Title:       "The Full Moon of 2021",
				URL:         "https://apod.nasa.gov/apod/image/2201/MoonstripsAnnotatedIG_crop1024.jpg",
			},
			expectedError: nil,
		}, {
			name:          "not time for ops",
			timeout:       1 * time.Nanosecond,
			url:           `https://api.nasa.gov/planetary/apod?api_key=` + os.Getenv(consts.Token) + `&date=2022-03-01`,
			expected:      &astro.AstroModel{},
			expectedError: errors.New(`Get "https://api.nasa.gov/planetary/apod?api_key=` + os.Getenv(consts.Token) + `&date=2022-03-01": context deadline exceeded`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			ctx, cancel := context.WithTimeout(context.Background(), tt.timeout)
			defer cancel()

			md, err := getMetadata(ctx, tt.url)
			if err == nil {
				require.NoError(t, err)

				require.Equal(t, tt.expected.Copyright, md.Copyright)
				require.Equal(t, tt.expected.Date, md.Date)
				require.Equal(t, tt.expected.Explanation, md.Explanation)
				require.Equal(t, tt.expected.MediaType, md.MediaType)
				require.Equal(t, tt.expected.Title, md.Title)
				require.Equal(t, tt.expected.URL, md.URL)
				require.Equal(t, tt.expected.ThumbURL, md.ThumbURL)
			} else {
				require.Equal(t, tt.expectedError.Error(), err.Error())
			}
		})
	}
}

func TestChooseURL(t *testing.T) {

	tests := []struct {
		name          string
		model         astro.AstroModel
		expectedUrl   string
		expectedError error
	}{
		{
			name: "IMAGE, HD_URL",
			model: astro.AstroModel{
				Date:      "2009-03-29",
				HDURL:     "https://apod.nasa.gov/apod/image/0903/sn94d_highz_big.jpg",
				MediaType: "image",
				URL:       "https://apod.nasa.gov/apod/image/0903/sn94d_highz.jpg",
			},
			expectedUrl:   "https://apod.nasa.gov/apod/image/0903/sn94d_highz_big.jpg",
			expectedError: nil,
		}, {
			name: "IMAGE, URL",
			model: astro.AstroModel{
				Date:      "2009-03-29",
				MediaType: "image",
				URL:       "https://apod.nasa.gov/apod/image/0903/sn94d_highz.jpg",
			},
			expectedUrl:   "https://apod.nasa.gov/apod/image/0903/sn94d_highz.jpg",
			expectedError: nil,
		}, {
			name: "IMAGE, HD_URL ONLY",
			model: astro.AstroModel{
				Date:      "2009-03-29",
				MediaType: "image",
				HDURL:     "https://apod.nasa.gov/apod/image/0903/sn94d_highz_big.jpg",
			},
			expectedUrl:   "https://apod.nasa.gov/apod/image/0903/sn94d_highz_big.jpg",
			expectedError: nil,
		}, {
			name: "VIDEO, THUMBNAIL",
			model: astro.AstroModel{
				Date:      "2017-07-31",
				MediaType: "video",
				ThumbURL:  "https://img.youtube.com/vi/rJzKDbnXyH0/0.jpg",
				URL:       "https://www.youtube.com/embed/rJzKDbnXyH0?rel=0",
			},
			expectedUrl:   "https://img.youtube.com/vi/rJzKDbnXyH0/0.jpg",
			expectedError: nil,
		}, {
			name: "ERR, NO THUMBNAIL, VIDEO ONLY",
			model: astro.AstroModel{
				Date:      "2017-07-31",
				MediaType: "video",
				URL:       "https://www.youtube.com/embed/rJzKDbnXyH0?rel=0",
			},
			expectedUrl:   "",
			expectedError: errors.New("invalid url"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			actual, err := chooseURL(&tt.model)

			if err == nil {
				require.Nil(t, err)
			} else {
				require.Equal(t, tt.expectedError.Error(), err.Error())
			}

			require.Equal(t, tt.expectedUrl, actual)
		})
	}
}

func TestStorePicture(t *testing.T) {
	pwd, err := os.Getwd()
	require.NoError(t, err)
	pwd += "/"

	tests := []struct {
		name     string
		date     string
		urlToPic string
		filename string
		sha256   [32]byte
	}{
		{
			name:     "OK, 2022-01-01",
			urlToPic: "https://apod.nasa.gov/apod/image/2201/MoonstripsAnnotatedIG_crop1024.jpg",
			filename: "2022-01-01.jpg",
			date:     "2022-01-01",
		},
		{
			name:     "OK, 2022-03-01",
			urlToPic: "https://apod.nasa.gov/apod/image/2203/DuelingBands_Dai_960.jpg",
			filename: "2022-03-01.jpg",
			date:     "2022-03-01",
		},
		{
			name:     "second insert, 2022-01-01",
			urlToPic: "https://apod.nasa.gov/apod/image/2201/MoonstripsAnnotatedIG_crop1024.jpg",
			filename: "2022-01-01.jpg",
			date:     "2022-01-01",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			raw, err := downloadPicture(context.Background(), tt.urlToPic)
			require.NoError(t, err)

			md := &astro.AstroModel{
				Date:      tt.date,
				RAW:       raw,
				URL:       tt.urlToPic,
				MediaType: "image",
			}

			tt.sha256 = sha256.Sum256(raw)

			err = storePicture(pwd, md)
			require.NoError(t, err)

			u, err := chooseURL(md)
			require.NoError(t, err)
			require.NotEmpty(t, u)

			ext, err := getPicExtension(u)
			require.NoError(t, err)
			require.NotEmpty(t, ext)

			f, err := os.Open(pwd + tt.filename)
			require.NoError(t, err)

			raw, err = io.ReadAll(f)
			require.NoError(t, err)

			require.Equal(t, tt.sha256, sha256.Sum256(raw))

			err = os.Remove(filepath.Join(pwd, tt.filename))
			require.NoError(t, err)
		})
	}
}

func TestSendResponse(t *testing.T) {

	tests := []struct {
		name     string
		status   int
		Message  string
		urls     []string
		expected []byte
	}{
		{
			name:     "200",
			status:   http.StatusOK,
			Message:  "",
			urls:     []string{"hehe1", "hehe2", "hehe3"},
			expected: []byte(`{"message":"","urls":["hehe1","hehe2","hehe3"]}` + "\n"),
		}, {
			name:     "400",
			status:   http.StatusOK,
			Message:  "some custom error",
			urls:     nil,
			expected: []byte(`{"message":"some custom error","urls":null}` + "\n"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()

			sendResponse(w, tt.status, tt.Message, tt.urls)

			body, err := io.ReadAll(w.Result().Body)
			require.NoError(t, err)
			require.Equal(t, string(tt.expected), string(body))

			require.Equal(t, tt.status, w.Code)
			require.Equal(t, "application/json", w.Header().Get("Content-Type"))
		})
	}
}

func TestGetPicExtension(t *testing.T) {
	tests := []struct {
		name        string
		payload     string
		expectedVal string
		ExpectedErr error
	}{
		{
			name:        "empty payload",
			payload:     "",
			expectedVal: "",
			ExpectedErr: errors.New("empty string"),
		},
		{
			name:        "bad payload",
			payload:     "kfs",
			expectedVal: "",
			ExpectedErr: errors.New("no file extension"),
		},
		{
			name:        "bad payload",
			payload:     "k.",
			expectedVal: "",
			ExpectedErr: errors.New("no file extension"),
		},
		{
			name:        "good payload",
			payload:     "k.h",
			expectedVal: "",
			ExpectedErr: errors.New("no file extension"),
		}, {
			name:        "good payload",
			payload:     "k.png",
			expectedVal: "png",
			ExpectedErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			res, err := getPicExtension(tt.payload)
			require.Equal(t, tt.expectedVal, res)

			if tt.ExpectedErr == nil {
				require.NoError(t, err)
			} else {
				require.Equal(t, tt.ExpectedErr.Error(), err.Error())
			}
		})
	}
}
