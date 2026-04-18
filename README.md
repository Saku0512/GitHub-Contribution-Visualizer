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

- `/` は Svelte フロントエンドへルーティング
- `/api` は Go バックエンドへルーティング

停止する場合:

```bash
docker compose down
```

## Current Prototype Scope

- GitHub ユーザー名を入力する
- Go API に送信する
- 仮の分析結果を返す
- Svelte でペルソナとビジュアル方針を表示する
- Traefik 経由でフロントと API を単一オリジンで公開する

## Next Ideas

- GitHub GraphQL API から contribution データを取得
- 活動傾向の集計ロジックを実装
- OpenAI などを用いたキャラクター生成プロンプトの作成
- 3D モデル / ドット絵生成パイプラインとの接続
