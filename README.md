# bonsAI

Raspberry Pi 5 (8GB) 上で動かす、ローカル LLM ベースの盆栽チャットです。  
現在の実装は `Preact UI + Go BFF + llama.cpp + Qdrant` を基本にしつつ、LLM やセンサーが未接続でも SSE の会話体験を止めないフェイルソフト構成になっています。

## 現在の構成

- `bonsAI_front`
  Preact + Vite のフロントエンドです。チャット UI、PWA、日英表示切り替え、`/memory` 記憶ビューを持ちます。
- `bonsAI_server`
  Go 製 BFF です。静的ファイル配信、SSE 中継、LLM 接続確認、センサー取得、翻訳、Qdrant 記憶検索/保存を担当します。
- `bonsAI_LLM`
  `llama.cpp` の起動補助スクリプトと設定テンプレートです。
- `docker-compose.yml`
  Raspberry Pi 向けの本番寄り構成です。`llm`、`memorydb`、`backend` を起動します。
- `docker-compose.mac.yml`
  Mac 上での軽量確認用です。`memorydb` と `backend` を起動し、LLM は未接続またはホスト側 LLM を利用します。

## 現在の実装でできること

- `/api/chat/stream` で SSE ストリーミング応答
- LLM 未接続時のデモ応答フォールバック
- センサー API 未接続時のデモセンサー値フォールバック
- 会話ターンの Qdrant 保存と、同一 `sessionId` 内での記憶再検索
- 応答後の非同期保存による SSE 経路の保護
- `/api/chat/translate` を使った日英表示切り替え
- `/memory` での保存記憶確認
- Service Worker を使った PWA 配信

## 重要な挙動

### SSE とフォールバック

標準の応答経路は次の通りです。

`Frontend -> Go BFF -> llama.cpp`

- LLM が利用できるときは、Go BFF が `llama.cpp` の OpenAI 互換 API へ接続し、そのストリームを UI に中継します。
- LLM が未設定、未起動、または接続失敗したときは、Go BFF がデモ応答へフォールバックします。
- どちらの場合も UI から見える応答形式は SSE のままです。

### センサー

- 現在の Compose 構成には専用センサーサービスは含まれていません。
- `BONSAI_SENSOR_API_URL` を設定しない場合、backend はデモセンサー値を返します。
- フロント側には「通常」「とても乾いた状態」「通知デモ」の切り替えがあり、UI の挙動確認ができます。

### 長期記憶

- Qdrant を有効にした場合、会話ターン単位で `user_message + assistant_message` を保存します。
- 検索は同一 `sessionId` に絞って実行されます。
- 保存は応答完了後に非同期で行うため、保存失敗がストリーミング応答を止めません。
- 現在の埋め込みは追加常駐プロセスを増やさない軽量なハッシュベース実装です。

## ポートと URL

### `make dev-up`

- Frontend: `http://127.0.0.1:5173`
- Backend: `http://127.0.0.1:8082`
- LLM API: `http://127.0.0.1:8081`

### `make docker-up`

- App: `http://127.0.0.1:8080`
- LLM health: `http://127.0.0.1:8081/health`
- Qdrant: `http://127.0.0.1:6333`

### `make docker-mac-up`

- App: `http://127.0.0.1:8080`
- Qdrant: `http://127.0.0.1:6333`

## 主な API

- `GET /healthz`
- `GET /api/system/status`
- `GET /api/sensors`
- `POST /api/chat/stream`
- `POST /api/chat/translate`
- `GET /api/memories`
- `GET /memory`

## 前提

### Raspberry Pi 本番想定

- Raspberry Pi 5 (8GB)
- Raspberry Pi OS 64-bit
- Docker Compose または systemd

### ローカル開発

- `go`
- `curl`
- `bun` または `npm`
- `cmake`
- `llama-server` 実行ファイル、または `bonsAI_LLM/scripts/bootstrap_llama_cpp.sh` でビルドした `llama.cpp`
- `docker compose` を使う場合は Docker Desktop / Docker Engine

`air` が入っていれば `make dev-up` 時に backend はホットリロード付きで起動します。未導入でも `go run .` で動作します。

## クイックスタート

### 1. LLM 用の設定を用意する

```bash
cp bonsAI_LLM/.env.example bonsAI_LLM/.env
```

既定では Hugging Face 上の GGUF を使います。ローカル GGUF を使う場合は `bonsAI_LLM/.env` の `BONSAI_LLM_MODEL_FILE` を設定してください。

### 2. `llama.cpp` をビルドする

```bash
cd bonsAI_LLM
./scripts/bootstrap_llama_cpp.sh
```

### 3. 開発スタックを起動する

```bash
cd ..
make dev-up
```

停止:

```bash
make dev-down
```

状態確認:

```bash
make dev-status
```

### `make dev-up` で起動するもの

- `llama.cpp` サーバー
- Go backend
- Vite 開発サーバー

注意:

- `make dev-up` では Qdrant は起動しません。
- そのため、記憶保存と `/memory` の実運用確認は `make docker-up` か、別途 Qdrant を起動した backend で行ってください。
- 初回はモデルの取得と LLM 起動待ちで時間がかかることがあります。

## Docker Compose

Raspberry Pi 寄りの一体構成です。frontend は backend コンテナに静的同梱され、`:8080` から配信されます。

### 1. Compose 用の設定を用意する

```bash
cp .env.example .env
```

主に次の値をこの `.env` で調整できます。

- `HF_TOKEN`
- `BONSAI_LLM_ALIAS`
- `BONSAI_LLM_MODEL`
- `BONSAI_LLM_HF_REPO`
- `BONSAI_LLM_HF_FILE`
- `BONSAI_MAC_LLM_CHAT_STREAM_URL`

### 2. 起動する

```bash
make docker-up
```

停止:

```bash
make docker-down
```

ログ:

```bash
make docker-logs
```

### `docker-compose.yml` の内容

- `llm`
  `llama.cpp` サーバーです。既定では `unsloth/gemma-4-E2B-it-GGUF` を参照します。
- `memorydb`
  Qdrant です。会話記憶を保存します。
- `backend`
  Go BFF です。静的 frontend を配信しつつ、LLM と Qdrant に接続します。

補足:

- 現在の Compose にはセンサーサービスは含まれていないため、backend はデモセンサーを使います。
- backend の記憶検索件数は既定で 3 件、ベクトル次元は 192 です。

## Mac 用 Docker 確認

Mac では軽量確認用に `docker-compose.mac.yml` を使えます。

起動:

```bash
make docker-mac-up
```

状態確認:

```bash
make docker-mac-status
```

ログ:

```bash
make docker-mac-logs
```

停止:

```bash
make docker-mac-down
```

この構成では:

- `memorydb` と `backend` のみを Docker で起動します
- `BONSAI_MAC_LLM_CHAT_STREAM_URL` が空なら、chat はデモ応答へフォールバックします
- Qdrant を含む SSE 経路と UI 配信の確認に向いています

ホスト側 LLM を使う場合の例:

```bash
BONSAI_MAC_LLM_CHAT_STREAM_URL=http://host.docker.internal:8081/v1/chat/completions \
make docker-mac-up
```

## 動作確認

### ヘルスチェック

```bash
curl -s http://127.0.0.1:8080/healthz
curl -s http://127.0.0.1:8080/api/system/status
```

### SSE チャット

```bash
curl -N -s -X POST http://127.0.0.1:8080/api/chat/stream \
  -H "Content-Type: application/json" \
  --data '{"sessionId":"readme-check","message":"今日は水やりが必要？","history":[]}'
```

### 記憶一覧

```bash
curl -s http://127.0.0.1:8080/api/memories?limit=5
```

ブラウザ確認:

- チャット画面: `http://127.0.0.1:8080/`
- 記憶ビュー: `http://127.0.0.1:8080/memory`

## 個別ビルドとテスト

リポジトリルートから実行します。

Frontend ビルド:

```bash
cd bonsAI_front && bun run build
```

Backend テスト:

```bash
cd bonsAI_server && go test ./...
```

## 補足

- Vite 単体開発時の API 向き先は `BONSAI_DEV_API_TARGET` で切り替えます。既定は `http://127.0.0.1:8082` です。
- backend は `runtime-config.js` を配信し、静的 frontend に `apiBase` を注入します。
- PWA の Service Worker は API と SSE をキャッシュ対象から外し、静的シェルのみをキャッシュします。

## 関連ドキュメント

- 全体設計: `./llm_document_raspi_only.md`
- LLM ノード詳細: `./bonsAI_LLM/README.md`
