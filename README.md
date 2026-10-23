# go-auth-api

Auth Foundation の認証 API を開発するための叩き台です。[go-api-template](https://github.com/OsolabAuth/go-api-template) の `main` (`1f94403`) から、HTTP サーバー・PostgreSQL 接続・JSON 共通処理を引き継いでいます。モジュール名は `github.com/OsolabAuth/go-auth-api` です。

現在は起動基盤とヘルスチェックまでを実装しています。ユーザー登録、ログイン、パスワード保存、トークン発行・検証、認可は未実装です。認証方式とデータモデルが決まってから追加します。テンプレートの tasks・notes API とサンプルテーブルは含めていません。

## ローカルで起動する

Docker Desktop（または Docker Engine と Compose）を用意してください。コマンド例は macOS・Linux・WSL 向けです。

```sh
cd authfoundation/go-auth-api
if [ ! -f .env ]; then cp .env.example .env; fi
docker compose up -d --build
```

API と PostgreSQL が起動します。API のホスト側ポートは `8081`、DB は `5433` で、Go テンプレートの初期設定と同時に使えます。両ポートはローカル PC のみに公開します。Compose のプロジェクト名は既定で `go-auth-api` となり、DB データはこのプロジェクトの `pgdata` ボリュームに保存されます。

```sh
curl -i http://localhost:8081/health
curl -i http://localhost:8081/ready
```

| エンドポイント | 正常時 | 用途 |
| --- | --- | --- |
| `GET /health` | `200 {"status":"ok"}` | API プロセスの応答確認 |
| `GET /ready` | `200 {"status":"ready"}` | PostgreSQL への接続確認。接続できなければ `503` |

API は起動時にも DB 接続を確認します。テーブルはまだないため、マイグレーションは不要です。

Docker Desktop からアプリ単位で起動・停止・ログ確認ができます。CLI では次のコマンドを使います。

```sh
docker compose logs -f api             # ログを見る
docker compose up -d --build api       # ソース変更後に再ビルドする
docker compose --profile tools down   # 停止・削除する。DB ボリュームは残る
```

この構成に自動リロードはありません。コード変更後は再ビルドしてください。

## Go をホスト上で実行する

Go（`go.mod` の指定バージョン以上）と make があれば、DB だけをコンテナで動かして開発できます。API を Compose で起動済みの場合は先に停止します。

```sh
docker compose stop api
make db-up
go mod download
make dev
```

`make dev` が `.env` を読み、ホスト上で `go run ./cmd/api` を実行します。ソース変更時は `Ctrl+C` で終了して再実行してください。直接 `go run` する場合、環境変数は自分で渡す必要があります。

`.env.example` の `DATABASE_URL` はホスト用です。Compose 内の API は `db:5432` を使う URL に差し替えます。`PORT` と `POSTGRES_PORT` は PC 側のポートで、コンテナ内はそれぞれ `8080` と `5432` です。DB のユーザー名・パスワード・DB 名はボリューム初期作成時に設定されます。

## テストとカバレッジ

```sh
make test
make coverage
```

DB なしで JSON 入出力、設定、ヘルスチェック、アクセスログをテストできます。PostgreSQL と合わせて確認する場合は、テスト DB の接続先を渡します。

```sh
TEST_DATABASE_URL='postgres://app:app_dev_password@localhost:5433/authdb?sslmode=disable' make coverage
```

統合テストは DB 接続時の `/ready` 成功と、接続プールを閉じた後の失敗を確認します。テーブルやデータは変更しません。`TEST_DATABASE_URL` が未指定の場合、このテストはスキップします。

カバレッジは一度のテスト実行から次の三つを生成します。

| レポート | 対象 |
| --- | --- |
| `coverage/index.html` | 全体 |
| `coverage/common.html` | `internal/common/` の共通処理 |
| `coverage/app.html` | 共通処理以外（起動・設定・DB 接続を含む） |

HTML をブラウザで開いて確認します。共通処理は現時点ではこのリポジトリ内で管理し、別モジュールにはしていません。カバレッジ処理と共通処理・設定のテストには、テンプレートの `example/split-coverage` / `example/test-coverage` を利用しています。

PR と `main` への push 時に GitHub Actions が vet、PostgreSQL を使うテスト、カバレッジ集計、Go ビルド、Compose での起動確認を実行します。結果は Actions のサマリーと `coverage-report` アーティファクトから確認できます。

## 構成と機能の追加先

| 場所 | 役割 |
| --- | --- |
| `cmd/api/main.go` | 設定・DB 接続・サーバー起動・終了 |
| `internal/app/app.go` | ルート登録、Request ID、アクセスログ、タイムアウト |
| `internal/common/json.go` | JSON 入出力と統一エラー形式 |
| `internal/config/` | 環境変数の読み込み |
| `internal/database/` | pgxpool の接続管理 |
| `migrations/` | 認証用スキーマを追加する場所 |
| `compose.yaml` | ローカル API・DB・マイグレーションツール |
| `docker/Dockerfile.prod` | 実行用 Docker イメージ |

認証機能を追加するときは `internal/resources/auth/` などを作り、`app.go` にルートを登録します。HTTP 入出力を `handler.go`、認証処理を必要に応じて `service.go`、SQL を `repository.go` に置く構成を想定しています。

JSON 入力はテンプレートと同じく `DisallowUnknownFields` で未定義フィールドを拒否し、本文を 1 MiB に制限します。DB スキーマの追加手順は [migrations/README.md](migrations/README.md) を参照してください。

## コマンド

| コマンド | 内容 |
| --- | --- |
| `make up` | API・DB をビルドして起動 |
| `make down` | Compose コンテナを停止・削除。DB データは保持 |
| `make logs` | API のログを表示 |
| `make db-up` | DB のみ起動 |
| `make dev` | `.env` を読み込み、ホストで API を起動 |
| `make test` | テストを実行 |
| `make coverage` | 共通処理・API 固有処理・全体のカバレッジを生成 |
| `make migrate-up` | SQL 追加後、未適用のマイグレーションを適用 |
| `make migrate-down` | 適用済みマイグレーションを一つ戻す |
| `make build` | `go-auth-api:local` イメージをビルド |

`.env` はローカル専用で、Git と Docker ビルド対象から除外します。デプロイ先は未設定です。
