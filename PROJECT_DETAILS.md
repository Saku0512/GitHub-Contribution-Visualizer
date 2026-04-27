# GitHub-Contribution-Visualizer 技術深掘りドキュメント

本ドキュメントでは、プロジェクトの内部実装、アルゴリズム、データフロー、および設計思想について、ソースコードレベルの詳細を解説します。

---

## 1. システムアーキテクチャとデータフロー

システムは「収集」「分析」「表現」の3つのフェーズで構成されています。

```mermaid
flowchart LR
    UB[User Browser] -- "1. Username" --> API[Go API Server]

    API -- "2. GraphQL Query" --> GH[:contentReference[oaicite:0]{index=0} API]
    API --> AE[Analysis Engine]

    AE -- "4. JSON Result" --> UB

    UB -- "5. Parameters" --> TJ[:contentReference[oaicite:1]{index=1} Engine]
    TJ --> AV[Render 3D Avatar]
```

1.  **収集**: `githubapi` クライアントが GitHub GraphQL API v4 を使用し、ユーザーのプロファイル、コントリビューションカレンダー、およびリポジトリごとの統計を1回のリクエストで取得。
2.  **分析**: `analysis` サービスが取得した「生データ（Snapshot）」を「意味のある指標（Metrics）」に変換。
3.  **表現**: 指標に基づいて「ペルソナ」と「3D生成用パラメータ（Spec）」を決定。フロントエンドでそれを3D形状に実体化。

---

## 2. バックエンドの実装詳細 (Go)

### 2.1 データ分析アルゴリズム (`internal/analysis/service.go`)

単なる合計値ではなく、時系列データからユーザーの「振る舞い」を抽出しています。

-   **Streak（継続日数）の計算**:
    コントリビューションカレンダーを走査し、連続して活動がある日数をカウント。`longestStreak`（過去1年の最大値）と `currentStreak`（直近の継続値）の両方を算出。
-   **Weekend Ratio（週末比率）**:
    `Saturday` と `Sunday` のコントリビューション合計を総計で割り、0.0〜1.0 の値で算出。これが高いと「趣味・副業プロジェクト型」と判定されやすくなります。
-   **Dominant Activity（主要活動）**:
    `Commits`, `PRs`, `Issues`, `Reviews` の中から最もカウントが多いものを抽出。これがアバターの「テーマカラー」の決定因子になります。

### 2.2 リポジトリ活動のマージロジック
GitHub API は活動タイプごとにリポジトリリストを返しますが、バックエンド側でこれらを同一リポジトリ名ごとに集約（Merge）し、総合的な「Top Repositories」を算出しています。

---

## 3. フロントエンドの実装詳細 (Svelte + Three.js)

### 3.1 3Dパラメータ・マッピング (`App.svelte`)

バックエンドから返された統計値を、3Dモデルの幾何学的数値に変換するための `buildAvatarSpec` 関数が肝となります。

| 3Dパーツ | 制御パラメータ | 計算式の意図 (Mapping) |
| :--- | :--- | :--- |
| **胴体 (Torso)** | `activeDays` (活動日数) | 年間を通した「安定感」を高さに反映。 |
| **頭部 (Head)** | `peakDay` (1日最大数) | 瞬発的に発揮できる「エネルギー量」を半径に反映。 |
| **フィン (Fins)** | `total` (総計) | 背中のパーツ数は、積み上げた「成果の量」を表す。 |
| **アンテナ** | `longestStreak` (継続) | 外部との接続や継続性をアンテナの本数と高さで表現。 |
| **サテライト** | `topRepositories` (数) | 周囲を回る浮遊オブジェクトは、活動の「広がり」を意味する。 |

**計算例 (Clamp関数の使用)**:
```javascript
// 活動日数(max 366)を 1.7〜2.45 の高さにマッピング
torsoHeight: clamp(1.6 + activeDays / 260, 1.7, 2.45)
```
これにより、極端な数値（数千コミットなど）があっても、3Dモデルが崩れないように制約（Clamp）をかけています。

### 3.2 3Dレンダリングエンジン (`AvatarPreview.svelte`)

-   **プロシージャル生成**: `Three.js` の `CapsuleGeometry` や `IcosahedronGeometry` を組み合わせてアバターを構築。静的なモデルを読み込むのではなく、JavaScriptコードがその場で「肉付け」を行います。
-   **マテリアルとライティング**: 
    - `MeshStandardMaterial` を使用し、金属光沢（metalness）と粗さ（roughness）を調整。
    - 3つの光源（Ambient, Key, Rim）を配置し、モデルの輪郭を強調する「リムライト」を当てることで、現代的なビジュアルに仕上げています。
-   **最適化**: `ResizeObserver` による自動リサイズ、および `onDestroy` でのメモリ解放（Geometry/Materialのdispose）を徹底し、ブラウザの負荷を軽減しています。

---

## 4. インフラとデプロイ設定

### 4.1 Traefik によるリバースプロキシ
`docker-compose.yml` で制御されており、フロントエンドとバックエンドの通信を同一ホスト・同一ポート上で解決します。

-   `frontend`: ポート `5173` (Vite)
-   `backend`: ポート `8080` (Go)
-   `Traefik`: ポート `8088` (Entrypoint)

これにより、CORS（Cross-Origin Resource Sharing）の問題を本番環境に近い形で回避しつつ、開発時は Vite のホットリロードの恩恵を受けることができます。

---

## 5. 今後の拡張性 (Roadmap)

1.  **キャラクター性格付け**: 分析結果を OpenAI API に渡し、アバターの性格や「ユーザーへのアドバイス」を生成する機能。
2.  **アニメーションの多様化**: `dominantActivity` が `Review` の場合は「見守るような動き」、`Commits` の場合は「活発な動き」など、活動傾向をモーションにも反映。
3.  **モデル書き出し**: 生成された3Dアバターを `.glb` または `.vrm` 形式でダウンロードできる機能。
