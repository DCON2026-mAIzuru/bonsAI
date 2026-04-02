# bonsAI

Raspberry Pi 5 (8GB) 単体で、センサー付きのローカル盆栽チャットを動かすためのプロトタイプです。

## 現在の構成

- `bonsAI_front`: 静的配信前提の Preact UI
- `bonsAI_server`: Go BFF。センサー API と LLM API を束ねる
- `bonsAI_LLM`: `llama.cpp` ベースの LLM ノード用ファイル

## LLM ノード

設計書 `llm_document_raspi_only.md` に合わせ、LLM は `llama.cpp` サーバーモードで localhost に立てる構成です。

- 目標モデル: Qwen2.5 3B Instruct
- バックエンド接続先: `http://127.0.0.1:8081/v1/chat/completions`
- バックエンドモデル名: `qwen2.5-3b`
- チャット応答の標準経路: Frontend -> Go BFF -> llama.cpp -> Qwen

Qwen が起動していない場合、チャットはデモ応答に落とさず接続エラーとして扱います。

詳細は `bonsAI_LLM/README.md` を参照してください。

## 一括起動

この Mac の開発用には Docker よりホスト起動のほうが `llama.cpp` と相性がよいので、1 コマンド起動スクリプトを用意しています。

```bash
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

起動後の URL:

- Frontend: `http://127.0.0.1:5173`
- Backend: `http://127.0.0.1:8082`
- Qwen API: `http://127.0.0.1:8081`

## Docker Compose

ラズパイ本番向けには Docker Compose でも起動できます。Compose 構成は次の 2 サービスです。

- `llm`: `llama.cpp` サーバー。Qwen2.5 3B GGUF を Hugging Face から取得して起動
- `backend`: Go BFF。フロント静的ファイルを内包して `:8080` で配信

起動:

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

アクセス先:

- App: `http://127.0.0.1:8080`
- LLM health: `http://127.0.0.1:8081/health`

補足:

- 初回起動時は Qwen モデルのダウンロードに時間がかかります
- Compose は Linux/arm64 のラズパイを主対象にしています
- センサー API ノードはまだ未実装なので、現状はセンサー値のみフォールバックです
