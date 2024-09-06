package app

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/gommon/log"
	"github.com/michimani/gotwi"
	"github.com/michimani/gotwi/tweet/managetweet"
	"github.com/michimani/gotwi/tweet/managetweet/types"
	"github.com/urfave/cli/v2"
)

func New() *cli.App {
	app := cli.NewApp()
	app.Name = "twdr"
	app.Description = "The reporter to Twitter of the amount of code written daily"
	app.Version = "0.0.1"

	app.Flags = []cli.Flag{
		&cli.BoolFlag{
			Name: "debug",
		},
		&cli.StringFlag{
			Name:     "tw-api-key",
			Required: true,
			EnvVars:  []string{"TW_API_KEY"},
		},
		&cli.StringFlag{
			Name:     "tw-api-secret-key",
			Required: true,
			EnvVars:  []string{"TW_API_SECRET"},
		},
		&cli.StringFlag{
			Name:     "tw-token",
			Required: true,
			EnvVars:  []string{"TW_TOKEN"},
		},
		&cli.StringFlag{
			Name:     "tw-token-secret",
			Required: true,
			EnvVars:  []string{"TW_TOKEN_SECRET"},
		},
		&cli.StringFlag{
			Name:     "wakatime-api-key",
			Required: true,
			EnvVars:  []string{"WAKATIME_API_KEY"},
		},
	}
	app.Before = func(c *cli.Context) error {
		if c.Bool("debug") {
			log.SetLevel(log.DEBUG)
		}
		return nil
	}
	app.Action = func(c *cli.Context) error {
		summary, err := GetTodaySummary(c)
		if err != nil {
			return err
		}

		tweet := "【自動投稿】\n今日は" + strings.Join(summary.TopLangs[:3], "、") + "のコードを" + strconv.FormatUint(summary.TotalHour, 10) + "時間" + strconv.FormatUint(summary.TotalMin, 10) + "分書きました！"

		err = PostTweet(c, tweet)
		if err != nil {
			return err
		}

		return nil
	}
	return app
}

func PostTweet(c *cli.Context, content string) error {
	api, err := GetTwitterApi(c)
	if err != nil {
		return err
	}

	tweet := &types.CreateInput{
		Text: gotwi.String(content),
	}

	_, err = managetweet.Create(context.Background(), api, tweet)
	if err != nil {
		return err
	}
	return nil
}

func GetTwitterApi(c *cli.Context) (*gotwi.Client, error) {
	os.Setenv("GOTWI_API_KEY", c.String("tw-api-key"))
	os.Setenv("GOTWI_API_KEY_SECRET", c.String("tw-api-secret-key"))
	in := &gotwi.NewClientInput{
		AuthenticationMethod: gotwi.AuthenMethodOAuth1UserContext,
		OAuthToken:           c.String("tw-token"),
		OAuthTokenSecret:     c.String("tw-token-secret"),
	}
	client, err := gotwi.NewClient(in)
	if err != nil {
		return nil, err
	}

	return client, nil
}

type (
	DurationResponse struct {
		Start    *time.Time             `json:"start"`
		End      *time.Time             `json:"end"`
		Timezone *string                `json:"timezone"`
		Data     []DurationResponseData `json:"data"`
	}

	DurationResponseData struct {
		Duration float64 `json:"duration"`
		Language string  `json:"language"`
		Project  string  `json:"project"`
		Time     float64 `json:"time"`
	}

	TodaySummary struct {
		TotalHour uint64
		TotalMin  uint64
		TopLangs  []string
	}

	Pair struct {
		Key   string
		Value float64
	}
	PairList []Pair
)

func (p PairList) Len() int           { return len(p) }
func (p PairList) Less(i, j int) bool { return p[i].Value < p[j].Value }
func (p PairList) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

func GetTodaySummary(c *cli.Context) (*TodaySummary, error) {
	httpClient := &http.Client{}

	yesterday := time.Now().AddDate(0, 0, -1)
	yesterdayStr := yesterday.Format("2006-01-02")

	req, err := http.NewRequest("GET", "https://wakatime.com/api/v1/users/current/durations?date="+yesterdayStr+"&slice_by=language", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(c.String("wakatime-api-key"))))

	res, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	bytes, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	var durationResponse DurationResponse
	err = json.Unmarshal(bytes, &durationResponse)
	if err != nil {
		return nil, err
	}

	durationSeconds := 0.0
	durationMap := make(map[string]float64)
	for _, data := range durationResponse.Data {
		if data.Language == "Other" {
			continue
		}
		if _, ok := durationMap[data.Language]; !ok {
			durationMap[data.Language] = 0
		}
		durationMap[data.Language] += data.Duration
		durationSeconds += data.Duration
	}

	pairList := make(PairList, 0, len(durationMap))
	for k, v := range durationMap {
		pairList = append(pairList, Pair{k, v})
	}
	sort.Sort(sort.Reverse(pairList))

	topLangs := make([]string, 0, len(pairList))
	for _, pair := range pairList {
		topLangs = append(topLangs, pair.Key)
	}

	duration := time.Duration(durationSeconds) * time.Second

	hours := uint64(duration.Hours())
	minutes := uint64(duration.Minutes()) % 60

	return &TodaySummary{
		TopLangs:  topLangs,
		TotalHour: hours,
		TotalMin:  minutes,
	}, nil
}
