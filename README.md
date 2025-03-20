# Google Calendarイベント集計ツール

このツールは、Google Calendarから特定期間内の指定されたイベントを取得し、その合計時間を集計するコマンドラインアプリケーションです。Go言語で実装されており、特定のカレンダーから指定されたイベント名に一致するイベントを抽出し、その合計時間を時間と分で表示します。

## 機能

- 指定された期間内のイベントを取得
- イベント名での検索（大文字小文字区別なし）
- 合計時間の計算と表示
- 一致したイベントの詳細リスト表示
- 利用可能なカレンダーの一覧表示
- 終日イベントは集計から除外
- トークンの自動更新機能（期限切れ時に自動的に更新または再認証）

## 前提条件

- Go言語の実行環境
- Googleアカウント
- Google Cloud Projectでの認証設定

## セットアップ

### 1. Google Cloud Projectの設定

1. [Google Cloud Console](https://console.cloud.google.com/)にアクセス
2. 新しいプロジェクトを作成
3. 「APIとサービス」→「ライブラリ」から「Google Calendar API」を有効化
4. 「認証情報」→「認証情報を作成」→「OAuth クライアントID」を選択
5. アプリケーションの種類として「デスクトップアプリ」を選択
6. リダイレクトURIとして `http://localhost:8080` を追加
7. 認証情報をダウンロードし、プロジェクトのルートディレクトリに `credentials.json` として保存

### 2. 依存パッケージのインストール

```bash
go get -u golang.org/x/oauth2/google
go get -u google.golang.org/api/calendar/v3
go get -u google.golang.org/api/option
```

### 3. アプリケーションの実行

初回実行時には、ブラウザが開いてGoogle認証が求められます。認証後、トークンが `token.json` に保存され、以降の実行では自動的に使用されます。

トークンの有効期限が切れた場合は、自動的に更新を試みます。リフレッシュトークンが有効であれば、ユーザーの操作なしに更新されます。リフレッシュトークンが無効または存在しない場合は、再度認証画面が表示されます。

## 使い方

### 基本的な使用法

```bash
go run main.go -start=YYYY-MM-DD -end=YYYY-MM-DD -name="イベント名" [-calendar="カレンダーID"]
```

### オプション

| オプション    | 説明                                     | 必須 | デフォルト値 |
|--------------|------------------------------------------|------|------------|
| `-start`     | 検索開始日（YYYY-MM-DD形式）              | はい  | なし        |
| `-end`       | 検索終了日（YYYY-MM-DD形式）              | はい  | なし        |
| `-name`      | 検索するイベント名                       | はい  | なし        |
| `-calendar`  | 使用するカレンダーID                     | いいえ | "primary"   |
| `-list`      | 利用可能なカレンダーの一覧を表示          | いいえ | false      |

### カレンダー一覧の表示

```bash
go run main.go -list
```

この機能を使うことで、利用可能なすべてのカレンダーのIDと名前を確認できます。

### 実行例

```bash
# プライマリカレンダーから「ミーティング」というイベントを2023年1月中で検索
go run main.go -start=2023-01-01 -end=2023-01-31 -name="ミーティング"

# 特定のカレンダーから「勉強会」というイベントを検索
go run main.go -start=2023-02-01 -end=2023-02-28 -name="勉強会" -calendar="example@gmail.com"
```

## 出力例

```
イベント 'ミーティング' の合計時間: 8時間 30分

一致したイベント一覧:
1. ミーティング (2023/01/05 10:00～2023/01/05 11:30) [1時間30分]
2. ミーティング (2023/01/12 14:00～2023/01/12 16:00) [2時間0分]
3. ミーティング (2023/01/19 09:00～2023/01/19 12:00) [3時間0分]
4. ミーティング (2023/01/26 13:00～2023/01/26 15:00) [2時間0分]
```

## 注意事項

- 初回実行時には、Googleアカウントへのアクセス許可が必要です
- タイムゾーンは「Asia/Tokyo」に設定されています
- イベント名は大文字小文字を区別せず完全一致で検索されます
- 終日イベントは集計対象から除外されます
- トークンは期限切れ時に自動的に更新されますが、長期間使用しなかった場合やGoogleの認証ポリシーが変更された場合は再認証が必要になることがあります
