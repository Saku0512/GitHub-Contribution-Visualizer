# GitHub-Contribution-Visualizer

自分のGitHubユーザー名を入れると、過去1年の活動（草）をAIが分析し、その人の「開発スタイル」に合わせたオリジナルの「3Dモデル」や「ドットキャラ」を生成するWebアプリ。

## Tech Stack

- Backend: Golang
- Frontend: Svelte + Vite

## Project Structure

```text
.
├── backend
│   ├── cmd/api
│   └── internal
│       └── analysis
├── frontend
│   └── src
└── README.md
```

## Local Development

### 1. Backend

```bash
cd backend
go run ./cmd/api
```

デフォルトでは `http://localhost:8080` で起動します。

### 2. Frontend

```bash
cd frontend
npm install
npm run dev
```

デフォルトでは `http://localhost:5173` で起動します。
Vite の proxy により `/api` はバックエンドへ転送されます。

## Docker Compose

Traefik をリバースプロキシとして使い、`frontend` と `backend` をまとめて起動できます。

```bash
docker compose up --build
```

デフォルトでは `http://localhost:8088` にアクセスしてください。
`80` 番を使いたい場合は、ルートに `.env` を置いて `TRAEFIK_PORT=80` を指定できます。
実データ分析を使うには `.env` に `GITHUB_TOKEN=...` を設定してください。
GitHub ログインも使う場合は `GITHUB_OAUTH_CLIENT_ID` と `GITHUB_OAUTH_CLIENT_SECRET` も設定してください。

`frontend` は `Vite dev server` として起動するので、`frontend/src` 以下を保存するとホットリロードされます。
依存を追加した場合も、`frontend` 起動時にコンテナ内で `npm install` が走るため、そのまま反映されます。

- `/` は Svelte フロントエンドへルーティング
- `/api` は Go バックエンドへルーティング

停止する場合:

```bash
docker compose down
```

## Current Prototype Scope

- GitHub ユーザー名を入力する
- GitHub でログインして自分のユーザー名を自動入力する
- Go API に送信する
- GitHub GraphQL API から過去1年の活動データを取得する
- 活動量・streak・週末比率・主要リポジトリを集計する
- Svelte でペルソナとビジュアル方針を表示する
- Traefik 経由でフロントと API を単一オリジンで公開する

## Next Ideas

- OpenAI などを用いたキャラクター生成プロンプトの作成
- 3D モデル / ドット絵生成パイプラインとの接続
