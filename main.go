package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

// getTokenFromWeb はウェブブラウザを通じてトークンを取得する
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	// ローカルサーバーを起動してリダイレクトを処理
	var authCode string
	codeCh := make(chan string)

	// リダイレクト先のハンドラーを設定
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code != "" {
			codeCh <- code
			w.Write([]byte("認証が完了しました。このページを閉じて構いません。"))
		} else {
			w.Write([]byte("認証コードが取得できませんでした。"))
		}
	})

	// 一時的なサーバーを起動
	server := &http.Server{Addr: ":8080"} // localhostの8080ポートで待機
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("サーバー起動エラー: %v", err)
		}
	}()

	// ブラウザでURL開く指示
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("ブラウザで以下のURLを開いてください:\n%v\n", authURL)

	// 認証コードを受け取る
	authCode = <-codeCh

	// サーバーを停止
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	server.Shutdown(ctx)

	// 認証コードを使ってトークンを取得
	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		log.Fatalf("トークンの取得に失敗しました: %v", err)
	}
	return tok
}

// tokenFromFile はファイルからトークンを読み込む
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

// getClient はOAuth2クライアントを取得する
func getClient(config *oauth2.Config) *http.Client {
	// トークンファイルのパス
	tokenFile := "token.json"
	tok, err := tokenFromFile(tokenFile)
	if err != nil {
		tok = getTokenFromWeb(config)
		saveToken(tokenFile, tok)
	}
	return config.Client(context.Background(), tok)
}

// saveToken はトークンをファイルに保存する
func saveToken(path string, token *oauth2.Token) {
	fmt.Printf("トークンを %s に保存します\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("トークンファイルの保存に失敗しました: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

// 利用可能なカレンダーを一覧表示する関数
func listCalendars(srv *calendar.Service) {
	calendarList, err := srv.CalendarList.List().Do()
	if err != nil {
		log.Fatalf("カレンダー一覧の取得に失敗しました: %v", err)
	}

	fmt.Println("利用可能なカレンダー一覧:")
	for i, item := range calendarList.Items {
		fmt.Printf("%d. %s (ID: %s)\n", i+1, item.Summary, item.Id)
	}
}

func main() {
	// コマンドライン引数の解析
	startDateStr := flag.String("start", "", "開始日（YYYY-MM-DD形式）")
	endDateStr := flag.String("end", "", "終了日（YYYY-MM-DD形式）")
	eventName := flag.String("name", "", "検索するイベント名")
	calendarID := flag.String("calendar", "primary", "カレンダーID（デフォルトは 'primary'）")
	isList := flag.Bool("list", false, "利用可能なカレンダーの一覧を表示")
	flag.Parse()

	// 引数の検証（calendarIDはデフォルト値があるので必須チェックしない）
	if !*isList && (*startDateStr == "" || *endDateStr == "" || *eventName == "") {
		fmt.Println("使用方法: go run main.go -start=YYYY-MM-DD -end=YYYY-MM-DD -name=イベント名 [-calendar=カレンダーID]")
		os.Exit(1)
	}

	// 認証設定
	ctx := context.Background()
	b, err := os.ReadFile("credentials.json")
	if err != nil {
		log.Fatalf("credentials.jsonの読み込みに失敗しました: %v", err)
	}

	config, err := google.ConfigFromJSON(b, calendar.CalendarReadonlyScope)
	if err != nil {
		log.Fatalf("OAuth2の設定に失敗しました: %v", err)
	}
	client := getClient(config)

	srv, err := calendar.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		log.Fatalf("Calendar APIの初期化に失敗しました: %v", err)
	}

	if *isList {
		listCalendars(srv)
		return
	}

	// 日付文字列をTime型に変換
	jst, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		log.Fatalf("タイムゾーンの読み込みに失敗しました: %v", err)
	}

	startDate, err := time.ParseInLocation("2006-01-02", *startDateStr, jst)
	if err != nil {
		log.Fatalf("開始日の解析に失敗しました: %v", err)
	}

	endDate, err := time.ParseInLocation("2006-01-02", *endDateStr, jst)
	if err != nil {
		log.Fatalf("終了日の解析に失敗しました: %v", err)
	}
	// 終了日の終わりまで含めるため、1日追加
	endDate = endDate.AddDate(0, 0, 1)

	// カレンダーイベントの取得（calendarIDを使用）
	events, err := srv.Events.List(*calendarID).
		TimeMin(startDate.Format(time.RFC3339)).
		TimeMax(endDate.Format(time.RFC3339)).
		SingleEvents(true).
		OrderBy("startTime").
		Do()
	if err != nil {
		log.Fatalf("イベントの取得に失敗しました: %v", err)
	}

	// イベントの集計
	var totalDuration time.Duration
	var matchedEvents []*calendar.Event

	for _, item := range events.Items {
		// 終日イベントはスキップ
		if item.Start.DateTime == "" {
			continue
		}

		// イベント名の大文字小文字を区別せずに比較
		if strings.EqualFold(item.Summary, *eventName) {
			startTime, err := time.Parse(time.RFC3339, item.Start.DateTime)
			if err != nil {
				log.Printf("開始時間の解析に失敗しました: %v", err)
				continue
			}

			endTime, err := time.Parse(time.RFC3339, item.End.DateTime)
			if err != nil {
				log.Printf("終了時間の解析に失敗しました: %v", err)
				continue
			}

			duration := endTime.Sub(startTime)
			totalDuration += duration
			matchedEvents = append(matchedEvents, item)
		}
	}

	// 結果の表示
	fmt.Printf("イベント '%s' の合計時間: %d時間 %d分\n\n", *eventName, int(totalDuration.Hours()), int(totalDuration.Minutes())%60)

	if len(matchedEvents) == 0 {
		fmt.Println("一致するイベントが見つかりませんでした。")
		return
	}

	fmt.Println("一致したイベント一覧:")
	for i, event := range matchedEvents {
		startTime, _ := time.Parse(time.RFC3339, event.Start.DateTime)
		endTime, _ := time.Parse(time.RFC3339, event.End.DateTime)
		duration := endTime.Sub(startTime)

		// 日本時間に変換して表示
		startTimeJST := startTime.In(jst)
		endTimeJST := endTime.In(jst)

		fmt.Printf("%d. %s (%s～%s) [%d時間%d分]\n",
			i+1,
			event.Summary,
			startTimeJST.Format("2006/01/02 15:04"),
			endTimeJST.Format("2006/01/02 15:04"),
			int(duration.Hours()),
			int(duration.Minutes())%60)
	}
}
